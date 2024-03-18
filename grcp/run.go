package grpc


import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	util "github.com/uninus-opensource/uninus-go-architect-common/log"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd/lb"
	"github.com/gorilla/handlers"
	"github.com/soheilhy/cmux"
	"go.elastic.co/apm/module/apmgrpc"

	"github.com/golang/protobuf/proto"
	middle "github.com/grpc-ecosystem/go-grpc-middleware"
	recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/sony/gobreaker"
	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)



var (
	ErrServer = errors.New("Internal server error")
	allowedOrigins []string
	rgbb *regexp.Regexp
)

const (
	DefaultGrpcClientRetry        = 3
	DefaultGrpcClientRetryTimeout = 30 * time.Second
	DefaultGrpcClientTimeout      = 60 * time.Second
)

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func getBBPattern() *regexp.Regexp {
	r, _ := regexp.Compile(`^(?:https?:\/\/)?(?:[^.]+\.)?uninus\.id(?::\d{1,5})?(\/.*)?`)
	return r
}

// Recovery return grpc server option with recovery handler
func Recovery(logger log.Logger) []grpc.ServerOption {
	handler := func(p interface{}) (err error) {
		logger.Log("panic", p)
		return ErrServer
	}
	opts := []recovery.Option{
		recovery.WithRecoveryHandler(handler),
	}
	serverOptions := []grpc.ServerOption{
		middle.WithUnaryServerChain(
			recovery.UnaryServerInterceptor(opts...),
		),
		middle.WithStreamServerChain(
			recovery.StreamServerInterceptor(opts...),
		)}
	return serverOptions
}

// BackoffRetries time.Duration
func BackoffRetries(timeout time.Duration) lb.Callback {
	expBackoff := ExponentialWithCappedMax(200*time.Millisecond, timeout)
	return func(n int, err error) (keepTrying bool, replacement error) {
		expV := expBackoff()
		fmt.Printf("Retry at %v still error: %v\n", expV, err)
		if expV == timeout || strings.Contains(err.Error(), "desc = transport is closing") || strings.Contains(err.Error(), "desc = OK: HTTP status code 200") || strings.Contains(err.Error(), "circuit breaker") {
			return false, nil
		}
		<-time.After(expV)
		return true, nil
	}
}

// DefaultCBSetting returns open circuit based on ratio for resilent CB
func DefaultCBSetting(name string, timeout time.Duration) gobreaker.Settings {
	return gobreaker.Settings{
		Name:          name,
		MaxRequests:   10,
		Interval:      2 * timeout,
		Timeout:       timeout,
		ReadyToTrip:   DefaultReadyToTrip,
		OnStateChange: DefaultOnStateChange,
		IsSuccessful:  DefaultIsSuccessful,
	}
}

// DefaultReadyToTrip returns open circuit based on ratio for resilent CB
func DefaultReadyToTrip(counts gobreaker.Counts) bool {
	fmt.Printf("CB Request: %v %v\n", counts.Requests, counts.TotalFailures)
	failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
	return counts.Requests >= 100 && failureRatio >= 0.6
}

// DefaultOnStateChange
func DefaultOnStateChange(name string, from gobreaker.State, to gobreaker.State) {
	fmt.Printf("CB State %s from %s to %s\n", name, from, to)
}

func DefaultIsSuccessful(err error) bool {
	if err == nil {
		return true
	}

	if strings.Contains(err.Error(), "desc = OK: HTTP status code 200") {
		return true
	}

	return false
}

// DefaultServerOptions returns grpc server option with validator and recovery
func DefaultServerOptions(logger log.Logger) []grpc.ServerOption {
	handler := func(p interface{}) (err error) {
		logger.Log("panic", p)
		return ErrServer
	}
	opts := []recovery.Option{
		recovery.WithRecoveryHandler(handler),
	}
	serverOptions := []grpc.ServerOption{
		middle.WithUnaryServerChain(
			validator.UnaryServerInterceptor(),
			recovery.UnaryServerInterceptor(opts...),
			apmgrpc.NewUnaryServerInterceptor(apmgrpc.WithRecovery()),
		),
		middle.WithStreamServerChain(
			validator.StreamServerInterceptor(),
			recovery.StreamServerInterceptor(opts...),
		)}
	return serverOptions
}

// Serve listen for client request
func Serve(address string, server *grpc.Server, logger log.Logger) {

	rgbb = getBBPattern()

	lis, err := net.Listen("tcp", address)
	if err != nil {
		logger.Log(util.LogError, err.Error())
		return
	}

	err = server.Serve(lis)
	if err != nil {
		logger.Log(util.LogError, err.Error())
		return
	}
}

// RegisterHTTPHandler register endpoint to http server
type RegisterHTTPHandler func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

// HTTPMiddleware is middleware for http handler
type HTTPMiddleware func(handler http.Handler) http.Handler

// HTTPOption are settings for http server
type HTTPOption struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type MuxHandler struct {
	RegisterTopPath   string
	Register          RegisterHTTPHandler
	SupHandlerTopPath string
	SupHandler        http.Handler
}

// DefaultHTTPOption return default option read and write timeout
func DefaultHTTPOption() HTTPOption {
	return HTTPOption{ReadTimeout: 60 * time.Second, WriteTimeout: 60 * time.Second}
}

// StreamHTTPOption return option no timeout
func StreamHTTPOption() HTTPOption {
	return HTTPOption{}
}

// ServeHTTP listen for http request
func ServeHTTP(grpcAddress, httpAddress string, register RegisterHTTPHandler,
	creds credentials.TransportCredentials, logger log.Logger, handlers ...HTTPMiddleware) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	rgbb = getBBPattern()

	mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true}))
	var opts []grpc.DialOption
	if creds == nil {
		opts = []grpc.DialOption{grpc.WithInsecure()}
	} else {
		opts = []grpc.DialOption{grpc.WithTransportCredentials(creds)}
	}
	err := register(ctx, mux, grpcAddress, opts)
	if err != nil {
		logger.Log(util.LogError, err.Error())
		return
	}

	var handler http.Handler
	handler = mux
	for _, hm := range handlers {
		handler = hm(handler)
	}

	http.ListenAndServe(httpAddress, handler)
}

// ServeGRPCAndHTTPWithAllowedOrigin listen to grpc and http request
func ServeGRPCAndHTTPWithAllowedOrigin(address, port, allowedOrigin string, grpcServer *grpc.Server,
	register RegisterHTTPHandler, creds credentials.TransportCredentials,
	cert tls.Certificate, logger log.Logger,
	option HTTPOption, handlers ...HTTPMiddleware) {

	if allowedOrigin != "" {
		allowedOrigins = strings.Split(allowedOrigin, ",")
	}

	if creds == nil {
		conn, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
		if err != nil {
			logger.Log(util.LogError, err.Error())
			return
		}
		mux := cmux.New(conn)

		// Match connections in order:
		// First grpc, then HTTP.
		grpcL := mux.Match(cmux.HTTP2(), cmux.HTTP2HeaderField("content-type", "application/grpc"))
		httpL := mux.Match(cmux.HTTP1Fast())

		ctx := context.Background()
		gwmux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true}))
		opts := []grpc.DialOption{grpc.WithInsecure()}
		err = register(ctx, gwmux, address, opts)
		if err != nil {
			logger.Log(util.LogError, err.Error())
			return
		}

		var handler http.Handler
		handler = gwmux
		for _, hm := range handlers {
			handler = hm(handler)
		}

		httpServer := &http.Server{
			Addr:    address,
			Handler: handler,
		}

		if option.ReadTimeout > 0 {
			httpServer.ReadTimeout = option.ReadTimeout
		}

		if option.WriteTimeout > 0 {
			httpServer.WriteTimeout = option.WriteTimeout
		}

		// Use the muxed listeners for your servers.
		go grpcServer.Serve(grpcL)
		go httpServer.Serve(httpL)

		// Start serving!
		mux.Serve()
	} else {
		ctx := context.Background()
		mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true}))
		opts := []grpc.DialOption{grpc.WithTransportCredentials(creds)}
		err := register(ctx, mux, address, opts)
		if err != nil {
			logger.Log(util.LogError, err.Error())
			return
		}

		var handler http.Handler
		handler = mux
		for _, hm := range handlers {
			handler = hm(handler)
		}

		srv := &http.Server{
			Addr:    address,
			Handler: grpcHandlerFunc(grpcServer, handler),
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{cert},
				NextProtos:   []string{"h2"},
			},
		}

		if option.ReadTimeout > 0 {
			srv.ReadTimeout = option.ReadTimeout
		}

		if option.WriteTimeout > 0 {
			srv.WriteTimeout = option.WriteTimeout
		}

		conn, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
		if err != nil {
			logger.Log(util.LogError, err.Error())
			return
		}

		err = srv.Serve(tls.NewListener(conn, srv.TLSConfig))
		if err != nil {
			logger.Log(util.LogError, err.Error())
			return
		}
	}
}

// ServeGRPCAndHTTP listen to grpc and http request
func ServeGRPCAndHTTP(address, port string, grpcServer *grpc.Server,
	register RegisterHTTPHandler, creds credentials.TransportCredentials,
	cert tls.Certificate, logger log.Logger,
	option HTTPOption, handlers ...HTTPMiddleware) {

	rgbb = getBBPattern()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	var tlsConfig *tls.Config
	if creds != nil {
		opts = []grpc.DialOption{grpc.WithTransportCredentials(creds)}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"h2"},
		}
	}

	ctx := context.Background()
	mux := runtime.NewServeMux(
		// runtime.WithForwardResponseOption(HttpSuccessHandler),
		// runtime.WithMarshalerOption("*", &EmptyMarshaler{}),
		// runtime.WithProtoErrorHandler(HttpErrorHandler),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true}),
	)

	err := register(ctx, mux, address, opts)
	if err != nil {
		logger.Log(util.LogError, err.Error())
		return
	}

	var handler http.Handler
	handler = mux
	for _, hm := range handlers {
		handler = hm(handler)
	}

	if grpcServer != nil {
		healthServer := health.NewServer()
		healthpb.RegisterHealthServer(grpcServer, healthServer)

		for name := range grpcServer.GetServiceInfo() {
			healthServer.SetServingStatus(
				name,
				healthpb.HealthCheckResponse_SERVING,
			)
		}

		healthServer.SetServingStatus(
			"",
			healthpb.HealthCheckResponse_SERVING,
		)
	}

	conn, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		logger.Log(util.LogError, err.Error())
		return
	}

	m := cmux.New(conn)
	grpcL := m.Match(cmux.HTTP2(), cmux.HTTP2HeaderField("content-type", "application/grpc"))
	httpL := m.Match(cmux.HTTP1Fast())
	httpServer := &http.Server{
		Addr:    address,
		Handler: handler,
	}

	if option.ReadTimeout > 0 {
		httpServer.ReadTimeout = option.ReadTimeout
	}

	if option.WriteTimeout > 0 {
		httpServer.WriteTimeout = option.WriteTimeout
	}

	// Use the muxed listeners for your servers.
	g := new(errgroup.Group)
	g.Go(func() error { return grpcServer.Serve(grpcL) })
	g.Go(func() error { return httpServer.Serve(httpL) })

	if tlsConfig != nil {
		srv := &http.Server{
			Addr:      address,
			Handler:   grpcHandlerFunc(grpcServer, handler),
			TLSConfig: tlsConfig,
		}

		if option.ReadTimeout > 0 {
			srv.ReadTimeout = option.ReadTimeout
		}

		if option.WriteTimeout > 0 {
			srv.WriteTimeout = option.WriteTimeout
		}

		conn, err := net.Listen("tcp", fmt.Sprintf(":%s", "8443"))
		if err != nil {
			logger.Log(util.LogError, err.Error())
			return
		}
		g.Go(func() error { return srv.Serve(tls.NewListener(conn, srv.TLSConfig)) })
	}

	g.Go(func() error { return m.Serve() })

	// Start serving!
	logger.Log("run server:", g.Wait())
}

// ServeGRPCAndHTTPMux listen to grpc and http request
func ServeGRPCAndHTTPMux(address, port string, grpcServer *grpc.Server,
	muxHandler MuxHandler, creds credentials.TransportCredentials,
	cert tls.Certificate, logger log.Logger,
	option HTTPOption, handlers ...HTTPMiddleware) {

	rgbb = getBBPattern()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	var tlsConfig *tls.Config
	if creds != nil {
		opts = []grpc.DialOption{grpc.WithTransportCredentials(creds)}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"h2"},
		}
	}

	ctx := context.Background()
	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true}),
		// runtime.WithForwardResponseOption(HttpSuccessHandler),
		// runtime.WithMarshalerOption("*", &EmptyMarshaler{}),
		// runtime.WithProtoErrorHandler(HttpErrorHandler),
	)

	if muxHandler.Register != nil {
		err := muxHandler.Register(ctx, mux, address, opts)
		if err != nil {
			logger.Log(util.LogError, err.Error())
			return
		}
	}

	var handler, mHandler http.Handler
	handler = mux
	mHandler = muxHandler.SupHandler
	for _, hm := range handlers {
		handler = hm(handler)
		mHandler = hm(mHandler)
	}

	if grpcServer != nil {
		healthServer := health.NewServer()
		healthpb.RegisterHealthServer(grpcServer, healthServer)

		for name := range grpcServer.GetServiceInfo() {
			healthServer.SetServingStatus(
				name,
				healthpb.HealthCheckResponse_SERVING,
			)
		}

		healthServer.SetServingStatus(
			"",
			healthpb.HealthCheckResponse_SERVING,
		)
	}

	topMux := http.NewServeMux()
	topMux.Handle(muxHandler.RegisterTopPath+"/", http.StripPrefix(muxHandler.RegisterTopPath, handler))
	topMux.Handle(muxHandler.SupHandlerTopPath+"/", http.StripPrefix(muxHandler.SupHandlerTopPath, mHandler))

	var httpServer *http.Server

	if creds != nil {
		httpServer = &http.Server{
			Addr:      address,
			Handler:   grpcHandlerFunc(grpcServer, topMux),
			TLSConfig: tlsConfig,
		}
	} else {
		httpServer = &http.Server{
			Addr:    address,
			Handler: topMux,
		}
	}

	if option.ReadTimeout > 0 {
		httpServer.ReadTimeout = option.ReadTimeout
	}

	if option.WriteTimeout > 0 {
		httpServer.WriteTimeout = option.WriteTimeout
	}

	conn, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		logger.Log(util.LogError, err.Error())
		return
	}

	if creds != nil {
		err = httpServer.Serve(tls.NewListener(conn, httpServer.TLSConfig))
		if err != nil {
			logger.Log(util.LogError, err.Error())
			return
		}
	} else {
		mux := cmux.New(conn)
		grpcL := mux.Match(cmux.HTTP2(), cmux.HTTP2HeaderField("content-type", "application/grpc"))
		httpL := mux.Match(cmux.HTTP1Fast())

		// Use the muxed listeners for your servers.
		go grpcServer.Serve(grpcL)
		go httpServer.Serve(httpL)
		// Start serving!
		mux.Serve()
	}

}

func HttpSuccessHandler(ctx context.Context, w http.ResponseWriter, p proto.Message) error {
	rsp := &Response{
		Status: "success",
		Data:   p,
	}
	rsp.Data = p
	buf, _ := json.Marshal(rsp)
	w.Write(buf)
	return nil
}

func HttpErrorHandler(ctx context.Context, mux *runtime.ServeMux, m runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-Type", "application/json")

	s, ok := status.FromError(err)
	if !ok {
		s = status.New(codes.Unknown, err.Error())
	}

	errMsg := strings.Split(s.Message(), ":")
	statusCode := strconv.Itoa(int(s.Code()))
	message := s.Message()
	if len(errMsg) == 2 {
		statusCode = errMsg[0]
		message = errMsg[1]
	}
	errorMsg := InfoMessage{
		Code:    statusCode,
		Message: message,
	}
	errMsgs := []InfoMessage{}
	errMsgs = append(errMsgs, errorMsg)

	resp := Response{
		Status:   "error",
		Messages: &errMsgs,
	}
	bs, _ := json.Marshal(&resp)
	w.Write(bs)
}

func ServeGRPCAndHTTPWithMaxCallRecvMsgSize(address, port string, grpcServer *grpc.Server,
	register RegisterHTTPHandler, creds credentials.TransportCredentials,
	cert tls.Certificate, logger log.Logger,
	option HTTPOption, maxReceiveMessageSize int, handlers ...HTTPMiddleware) {

	rgbb = getBBPattern()
	if creds == nil {
		conn, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
		if err != nil {
			logger.Log(util.LogError, err.Error())
			return
		}
		mux := cmux.New(conn)

		// Match connections in order:
		// First grpc, then HTTP.
		grpcL := mux.Match(cmux.HTTP2(), cmux.HTTP2HeaderField("content-type", "application/grpc"))
		httpL := mux.Match(cmux.HTTP1Fast())

		ctx := context.Background()
		gwmux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true}))
		opts := []grpc.DialOption{grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxReceiveMessageSize), grpc.MaxCallSendMsgSize(maxReceiveMessageSize))}
		err = register(ctx, gwmux, address, opts)
		if err != nil {
			logger.Log(util.LogError, err.Error())
			return
		}

		var handler http.Handler
		handler = gwmux
		for _, hm := range handlers {
			handler = hm(handler)
		}

		if grpcServer != nil {
			healthServer := health.NewServer()
			healthpb.RegisterHealthServer(grpcServer, healthServer)

			for name := range grpcServer.GetServiceInfo() {
				healthServer.SetServingStatus(
					name,
					healthpb.HealthCheckResponse_SERVING,
				)
			}

			healthServer.SetServingStatus(
				"",
				healthpb.HealthCheckResponse_SERVING,
			)
		}

		httpServer := &http.Server{
			Addr:    address,
			Handler: handler,
		}

		if option.ReadTimeout > 0 {
			httpServer.ReadTimeout = option.ReadTimeout
		}

		if option.WriteTimeout > 0 {
			httpServer.WriteTimeout = option.WriteTimeout
		}

		// Use the muxed listeners for your servers.
		go grpcServer.Serve(grpcL)
		go httpServer.Serve(httpL)

		// Start serving!
		mux.Serve()
	} else {
		ctx := context.Background()
		mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true}))
		opts := []grpc.DialOption{grpc.WithTransportCredentials(creds), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxReceiveMessageSize), grpc.MaxCallSendMsgSize(maxReceiveMessageSize))}
		err := register(ctx, mux, address, opts)
		if err != nil {
			logger.Log(util.LogError, err.Error())
			return
		}

		var handler http.Handler
		handler = mux
		for _, hm := range handlers {
			handler = hm(handler)
		}

		if grpcServer != nil {
			healthServer := health.NewServer()
			healthpb.RegisterHealthServer(grpcServer, healthServer)

			for name := range grpcServer.GetServiceInfo() {
				healthServer.SetServingStatus(
					name,
					healthpb.HealthCheckResponse_SERVING,
				)
			}

			healthServer.SetServingStatus(
				"",
				healthpb.HealthCheckResponse_SERVING,
			)
		}

		srv := &http.Server{
			Addr:    address,
			Handler: grpcHandlerFunc(grpcServer, handler),
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{cert},
				NextProtos:   []string{"h2"},
			},
		}

		if option.ReadTimeout > 0 {
			srv.ReadTimeout = option.ReadTimeout
		}

		if option.WriteTimeout > 0 {
			srv.WriteTimeout = option.WriteTimeout
		}

		conn, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
		if err != nil {
			logger.Log(util.LogError, err.Error())
			return
		}

		err = srv.Serve(tls.NewListener(conn, srv.TLSConfig))
		if err != nil {
			logger.Log(util.LogError, err.Error())
			return
		}
	}
}

func ServeGRPCHandler(address, port string, grpcServer *grpc.Server, creds credentials.TransportCredentials,
	cert tls.Certificate, logger log.Logger,
	option HTTPOption, handlers ...HTTPMiddleware) {

	rgbb = getBBPattern()
	if creds == nil {
		conn, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
		if err != nil {
			logger.Log(util.LogError, err.Error())
			return
		}
		mux := cmux.New(conn)

		// Match connections in order:
		// First grpc, then HTTP.
		grpcL := mux.Match(cmux.HTTP2(), cmux.HTTP2HeaderField("content-type", "application/grpc"))
		httpL := mux.Match(cmux.HTTP1Fast())

		gwmux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true}))

		var handler http.Handler
		handler = gwmux
		for _, hm := range handlers {
			handler = hm(handler)
		}

		if grpcServer != nil {
			healthServer := health.NewServer()
			healthpb.RegisterHealthServer(grpcServer, healthServer)

			for name := range grpcServer.GetServiceInfo() {
				healthServer.SetServingStatus(
					name,
					healthpb.HealthCheckResponse_SERVING,
				)
			}

			healthServer.SetServingStatus(
				"",
				healthpb.HealthCheckResponse_SERVING,
			)
		}

		httpServer := &http.Server{
			Addr:    address,
			Handler: handler,
		}

		if option.ReadTimeout > 0 {
			httpServer.ReadTimeout = option.ReadTimeout
		}

		if option.WriteTimeout > 0 {
			httpServer.WriteTimeout = option.WriteTimeout
		}

		// Use the muxed listeners for your servers.
		go grpcServer.Serve(grpcL)
		go httpServer.Serve(httpL)

		// Start serving!
		mux.Serve()
	} else {
		mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true}))

		var tlsConfig *tls.Config
		if creds != nil {
			tlsConfig = &tls.Config{
				Certificates: []tls.Certificate{cert},
				NextProtos:   []string{"h2"},
			}
		}

		var handler http.Handler
		handler = mux
		for _, hm := range handlers {
			handler = hm(handler)
		}

		if grpcServer != nil {
			healthServer := health.NewServer()
			healthpb.RegisterHealthServer(grpcServer, healthServer)

			for name := range grpcServer.GetServiceInfo() {
				healthServer.SetServingStatus(
					name,
					healthpb.HealthCheckResponse_SERVING,
				)
			}

			healthServer.SetServingStatus(
				"",
				healthpb.HealthCheckResponse_SERVING,
			)
		}

		conn, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
		if err != nil {
			logger.Log(util.LogError, err.Error())
			return
		}

		m := cmux.New(conn)
		grpcL := m.Match(cmux.HTTP2(), cmux.HTTP2HeaderField("content-type", "application/grpc"))

		// Use the muxed listeners for your servers.
		g := new(errgroup.Group)
		g.Go(func() error { return grpcServer.Serve(grpcL) })

		if tlsConfig != nil {
			srv := &http.Server{
				Addr:      address,
				Handler:   grpcHandlerFunc(grpcServer, handler),
				TLSConfig: tlsConfig,
			}

			if option.ReadTimeout > 0 {
				srv.ReadTimeout = option.ReadTimeout
			}

			if option.WriteTimeout > 0 {
				srv.WriteTimeout = option.WriteTimeout
			}

			conn, err := net.Listen("tcp", fmt.Sprintf(":%s", "8443"))
			if err != nil {
				logger.Log(util.LogError, err.Error())
				return
			}
			g.Go(func() error { return srv.Serve(tls.NewListener(conn, srv.TLSConfig)) })
		}

		g.Go(func() error { return m.Serve() })
	}
}

func ServeGRPCAndHTTPHandler(address, port string, grpcServer *grpc.Server,
	register RegisterHTTPHandler, creds credentials.TransportCredentials,
	cert tls.Certificate, logger log.Logger,
	option HTTPOption, handlers ...HTTPMiddleware) {

	rgbb = getBBPattern()
	if creds == nil {
		conn, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
		if err != nil {
			logger.Log(util.LogError, err.Error())
			return
		}
		mux := cmux.New(conn)

		// Match connections in order:
		// First grpc, then HTTP.
		grpcL := mux.Match(cmux.HTTP2(), cmux.HTTP2HeaderField("content-type", "application/grpc"))
		httpL := mux.Match(cmux.HTTP1Fast())

		gwmux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true}))

		var handler http.Handler
		handler = gwmux
		for _, hm := range handlers {
			handler = hm(handler)
		}

		if grpcServer != nil {
			healthServer := health.NewServer()
			healthpb.RegisterHealthServer(grpcServer, healthServer)

			for name := range grpcServer.GetServiceInfo() {
				healthServer.SetServingStatus(
					name,
					healthpb.HealthCheckResponse_SERVING,
				)
			}

			healthServer.SetServingStatus(
				"",
				healthpb.HealthCheckResponse_SERVING,
			)
		}

		httpServer := &http.Server{
			Addr:    address,
			Handler: handler,
		}

		if option.ReadTimeout > 0 {
			httpServer.ReadTimeout = option.ReadTimeout
		}

		if option.WriteTimeout > 0 {
			httpServer.WriteTimeout = option.WriteTimeout
		}

		// Use the muxed listeners for your servers.
		go grpcServer.Serve(grpcL)
		go httpServer.Serve(httpL)

		// Start serving!
		mux.Serve()
	} else {
		rgbb = getBBPattern()
		opts := []grpc.DialOption{grpc.WithInsecure()}
		var tlsConfig *tls.Config
		if creds != nil {
			opts = []grpc.DialOption{grpc.WithTransportCredentials(creds)}
			tlsConfig = &tls.Config{
				Certificates: []tls.Certificate{cert},
				NextProtos:   []string{"h2"},
			}
		}

		mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true}))

		ctx := context.Background()
		err := register(ctx, mux, address, opts)
		if err != nil {
			logger.Log(util.LogError, err.Error())
			return
		}

		var handler http.Handler
		handler = mux
		for _, hm := range handlers {
			handler = hm(handler)
		}

		if grpcServer != nil {
			healthServer := health.NewServer()
			healthpb.RegisterHealthServer(grpcServer, healthServer)

			for name := range grpcServer.GetServiceInfo() {
				healthServer.SetServingStatus(
					name,
					healthpb.HealthCheckResponse_SERVING,
				)
			}

			healthServer.SetServingStatus(
				"",
				healthpb.HealthCheckResponse_SERVING,
			)
		}

		conn, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
		if err != nil {
			logger.Log(util.LogError, err.Error())
			return
		}

		m := cmux.New(conn)
		grpcL := m.Match(cmux.HTTP2(), cmux.HTTP2HeaderField("content-type", "application/grpc"))
		httpL := m.Match(cmux.HTTP1Fast())
		httpServer := &http.Server{
			Addr:    address,
			Handler: handler,
		}

		if option.ReadTimeout > 0 {
			httpServer.ReadTimeout = option.ReadTimeout
		}

		if option.WriteTimeout > 0 {
			httpServer.WriteTimeout = option.WriteTimeout
		}

		// Use the muxed listeners for your servers.
		g := new(errgroup.Group)
		g.Go(func() error { return grpcServer.Serve(grpcL) })
		g.Go(func() error { return httpServer.Serve(httpL) })

		if tlsConfig != nil {
			srv := &http.Server{
				Addr:      address,
				Handler:   grpcHandlerFunc(grpcServer, handler),
				TLSConfig: tlsConfig,
			}

			if option.ReadTimeout > 0 {
				srv.ReadTimeout = option.ReadTimeout
			}

			if option.WriteTimeout > 0 {
				srv.WriteTimeout = option.WriteTimeout
			}

			conn, err := net.Listen("tcp", fmt.Sprintf(":%s", "8443"))
			if err != nil {
				logger.Log(util.LogError, err.Error())
				return
			}
			g.Go(func() error { return srv.Serve(tls.NewListener(conn, srv.TLSConfig)) })
		}

		g.Go(func() error { return m.Serve() })

		// Start serving!
		logger.Log("run server:", g.Wait())
	}
}

func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	})
}

// HealthCheckHandler returns http 200 for root path
func HealthCheckHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
		} else {
			handler.ServeHTTP(w, r)
		}
	})
}

// CORSHandler enables cross-origin resource sharing
func CORSHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			if rgbb == nil {
				rgbb = getBBPattern()
			}

			if rgbb.MatchString(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
					headers := []string{"Content-Type", "Accept", "Authorization"}
					w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ","))
					methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE"}
					w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
					w.Header().Set("Access-Control-Max-Age", "86400")
					return
				}
			} else {
				w.WriteHeader(http.StatusForbidden)
				return
			}
		}
		handler.ServeHTTP(w, r)
	})
}

// DefaultHTTPHandler specifies default http handler
func DefaultHTTPHandler(handler http.Handler) http.Handler {
	handler = LogRequestHandler(handler)
	handler = handlers.CompressHandler(handler)
	handler = CORSHandler(handler)
	handler = HealthCheckHandler(handler)
	return handler
}

// CORSHandlerWithAllowedOrigin enables cross-origin resource sharing
func CORSHandlerWithAllowedOrigin(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			if contains(allowedOrigins, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
					headers := []string{"Content-Type", "Accept", "Authorization"}
					w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ","))
					methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE"}
					w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
					w.Header().Set("Access-Control-Max-Age", "86400")
					return
				}
			} else {
				w.WriteHeader(http.StatusForbidden)
				return
			}
		}
		handler.ServeHTTP(w, r)
	})
}

// DefaultHTTPHandlerWithAllowedOrigin specifies default http handler
func DefaultHTTPHandlerWithAllowedOrigin(handler http.Handler) http.Handler {
	handler = LogRequestHandler(handler)
	handler = handlers.CompressHandler(handler)
	handler = CORSHandlerWithAllowedOrigin(handler)
	handler = HealthCheckHandler(handler)
	return handler
}

func LogRequestHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := util.LogRequestClient(w, r)
		fmt.Println(util.LogReq, "Log Request Client", util.LogInfo, fmt.Sprintf("%+v", resp))
		handler.ServeHTTP(w, r)
	})
}
