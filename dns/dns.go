package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc/grpclog"

	"google.golang.org/grpc/resolver"
)

// EnableSRVLookups controls whether the DNS resolver attempts to fetch gRPCLB
// addresses from SRV records.  Must not be changed after init time.
var EnableSRVLookups = false

var (
	dnsCache           = make(map[string][]resolver.Address)
	dnsCacheLock       sync.Mutex
	stopClearCache     context.Context
	stopClearCacheFunc context.CancelFunc
)

const CACHE_KEY_DNS = "dns"

func init() {
	resolver.Register(NewBuilder())
	// Run clear cached in the backgroud with go routine
	stopClearCache, stopClearCacheFunc = context.WithCancel(context.Background())
	go clearCached(stopClearCache)
}

const (
	defaultPort       = "443"
	defaultDNSSvrPort = "53"
	golang            = "GO"
	// txtPrefix is the prefix string to be prepended to the host name for txt record lookup.
	txtPrefix = "_grpc_config."
	// In DNS, service config is encoded in a TXT record via the mechanism
	// described in RFC-1464 using the attribute name grpc_config.
	txtAttribute = "grpc_config="
)

var (
	errMissingAddr = errors.New("dns resolver: missing address")

	// Addresses ending with a colon that is supposed to be the separator
	// between host and port is not allowed.  E.g. "::" is a valid address as
	// it is an IPv6 address (host only) and "[::]:" is invalid as it ends with
	// a colon as the host and port separator
	errEndsWithColon = errors.New("dns resolver: missing port after port-separator colon")
)

var (
	defaultResolver netResolver = net.DefaultResolver
	// To prevent excessive re-resolution, we enforce a rate limit on DNS
	// resolution requests.
	minDNSResRate = 30 * time.Second
)

var customAuthorityDialler = func(authority string) func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		var dialer net.Dialer
		return dialer.DialContext(ctx, network, authority)
	}
}

var customAuthorityResolver = func(authority string) (netResolver, error) {
	host, port, err := parseTarget(authority, defaultDNSSvrPort)
	if err != nil {
		return nil, err
	}

	authorityWithPort := net.JoinHostPort(host, port)

	return &net.Resolver{
		PreferGo: true,
		Dial:     customAuthorityDialler(authorityWithPort),
	}, nil
}

// NewBuilder creates a dnsBuilder which is used to factory DNS resolvers.
func NewBuilder() resolver.Builder {
	return &dnsBuilder{}
}

type dnsBuilder struct{}

// Build creates and starts a DNS resolver that watches the name resolution of the target.
func (b *dnsBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {

	host, port, err := parseTarget(target.Endpoint(), defaultPort)
	if err != nil {
		return nil, err
	}

	// IP address.
	if ipAddr, ok := formatIP(host); ok {
		addr := []resolver.Address{{Addr: ipAddr + ":" + port}}
		cc.UpdateState(resolver.State{Addresses: addr})
		return deadResolver{}, nil
	}

	// DNS address (non-IP).
	ctx, cancel := context.WithCancel(context.Background())
	d := &dnsResolver{
		host:                 host,
		port:                 port,
		ctx:                  ctx,
		cancel:               cancel,
		cc:                   cc,
		rn:                   make(chan struct{}, 1),
		disableServiceConfig: opts.DisableServiceConfig,
		cacheKey:             target.Endpoint(),
	}

	if target.URL.Host == "" {
		d.resolver = defaultResolver
	} else {
		d.resolver, err = customAuthorityResolver(target.URL.Host)
		if err != nil {
			return nil, err
		}
	}

	d.wg.Add(1)
	go d.watcher()
	d.ResolveNow(resolver.ResolveNowOptions{})
	return d, nil
}

// Scheme returns the naming scheme of this resolver builder, which is "dns".
func (b *dnsBuilder) Scheme() string {
	return CACHE_KEY_DNS
}

type netResolver interface {
	LookupHost(ctx context.Context, host string) (addrs []string, err error)
	LookupSRV(ctx context.Context, service, proto, name string) (cname string, addrs []*net.SRV, err error)
	LookupTXT(ctx context.Context, name string) (txts []string, err error)
}

// deadResolver is a resolver that does nothing.
type deadResolver struct{}

func (deadResolver) ResolveNow(resolver.ResolveNowOptions) {}

func (deadResolver) Close() {}

// dnsResolver watches for the name resolution update for a non-IP target.
type dnsResolver struct {
	host     string
	port     string
	resolver netResolver
	ctx      context.Context
	cancel   context.CancelFunc
	cc       resolver.ClientConn
	// rn channel is used by ResolveNow() to force an immediate resolution of the target.
	rn chan struct{}
	// wg is used to enforce Close() to return after the watcher() goroutine has finished.
	// Otherwise, data race will be possible. [Race Example] in dns_resolver_test we
	// replace the real lookup functions with mocked ones to facilitate testing.
	// If Close() doesn't wait for watcher() goroutine finishes, race detector sometimes
	// will warns lookup (READ the lookup function pointers) inside watcher() goroutine
	// has data race with replaceNetFunc (WRITE the lookup function pointers).
	wg                   sync.WaitGroup
	disableServiceConfig bool

	cacheKey string
}

// ResolveNow invoke an immediate resolution of the target that this dnsResolver watches.
func (d *dnsResolver) ResolveNow(resolver.ResolveNowOptions) {
	select {
	case d.rn <- struct{}{}:
	default:
	}
}

// Close closes the dnsResolver.
func (d *dnsResolver) Close() {
	d.cancel()
	d.wg.Wait()
}

func (d *dnsResolver) watcher() {
	defer d.wg.Done()
	for {
		select {
		case <-d.ctx.Done():
			stopClearCacheFunc()
			return
		case <-d.rn:
		}

		state := d.lookup()
		d.cc.UpdateState(*state)
	}
}

func (d *dnsResolver) lookupCached() []resolver.Address {
	dnsCacheLock.Lock()
	cachedAddrs, found := dnsCache[d.cacheKey]
	dnsCacheLock.Unlock()
	if found {
		return cachedAddrs
	}

	return nil
}

func clearCached(ctx context.Context) {
	ticker := time.NewTicker(minDNSResRate)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dnsCacheLock.Lock()
			// Delete all keys in the map
			for key := range dnsCache {
				delete(dnsCache, key)
			}
			dnsCacheLock.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

func (d *dnsResolver) lookupHost() []resolver.Address {

	var newAddrs []resolver.Address
	addrs, err := d.resolver.LookupHost(d.ctx, d.host)
	if err != nil {
		grpclog.Warningf("grpc: failed dns A record lookup due to %v.\n", err)
		return nil
	}
	for _, a := range addrs {
		a, ok := formatIP(a)
		if !ok {
			grpclog.Errorf("grpc: failed IP parsing due to %v.\n", err)
			continue
		}
		addr := a + ":" + d.port
		newAddrs = append(newAddrs, resolver.Address{Addr: addr})
	}

	dnsCacheLock.Lock()
	dnsCache[d.cacheKey] = newAddrs
	dnsCacheLock.Unlock()

	return newAddrs
}

func (d *dnsResolver) lookup() *resolver.State {

	addr := d.lookupCached()
	if addr != nil {
		state := &resolver.State{
			Addresses: addr,
		}
		return state
	}

	addr = d.lookupHost()
	state := &resolver.State{
		Addresses: addr,
	}
	return state
}

// formatIP returns ok = false if addr is not a valid textual representation of an IP address.
// If addr is an IPv4 address, return the addr and ok = true.
// If addr is an IPv6 address, return the addr enclosed in square brackets and ok = true.
func formatIP(addr string) (addrIP string, ok bool) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return "", false
	}
	if ip.To4() != nil {
		return addr, true
	}
	return "[" + addr + "]", true
}

// parseTarget takes the user input target string and default port, returns formatted host and port info.
// If target doesn't specify a port, set the port to be the defaultPort.
// If target is in IPv6 format and host-name is enclosed in square brackets, brackets
// are stripped when setting the host.
// examples:
// target: "www.google.com" defaultPort: "443" returns host: "www.google.com", port: "443"
// target: "ipv4-host:80" defaultPort: "443" returns host: "ipv4-host", port: "80"
// target: "[ipv6-host]" defaultPort: "443" returns host: "ipv6-host", port: "443"
// target: ":80" defaultPort: "443" returns host: "localhost", port: "80"
func parseTarget(target, defaultPort string) (host, port string, err error) {
	if target == "" {
		return "", "", errMissingAddr
	}
	if ip := net.ParseIP(target); ip != nil {
		// target is an IPv4 or IPv6(without brackets) address
		return target, defaultPort, nil
	}
	if host, port, err = net.SplitHostPort(target); err == nil {
		if port == "" {
			// If the port field is empty (target ends with colon), e.g. "[::1]:", this is an error.
			return "", "", errEndsWithColon
		}
		// target has port, i.e ipv4-host:port, [ipv6-host]:port, host-name:port
		if host == "" {
			// Keep consistent with net.Dial(): If the host is empty, as in ":80", the local system is assumed.
			host = "localhost"
		}
		return host, port, nil
	}
	if host, port, err = net.SplitHostPort(target + ":" + defaultPort); err == nil {
		// target doesn't have port
		return host, port, nil
	}
	return "", "", fmt.Errorf("invalid target address %v, error info: %v", target, err)
}
