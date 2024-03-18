package grpc

import (
	"errors"
	"io"
	"strings"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	stdopentracing "github.com/opentracing/opentracing-go"
	"github.com/sony/gobreaker"
	"golang.org/x/net/context"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/resolver"

	"github.com/uninus-opensource/uninus-go-architect-common/dns"
	ulog "github.com/uninus-opensource/uninus-go-architect-common/log"
	"go.elastic.co/apm/module/apmgrpc"

	"google.golang.org/grpc"
)

// ClientOption stores grpc client options
type ClientOption struct {
	//Timeout for circuit breaker
	Timeout time.Duration
	//Number of retry
	Retry int
	//Timeout for retry
	RetryTimeout time.Duration
	// ...
	MaxCallRecvMsgSize int
}

func grpcConnection(address string, creds credentials.TransportCredentials) (*grpc.ClientConn, error) {
	var conn *grpc.ClientConn
	var err error
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithUnaryInterceptor(apmgrpc.NewUnaryClientInterceptor()))
	if strings.Contains(address, "dns:///") {
		resolver.Register(dns.NewBuilder())
		opts = append(opts, grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`))
	}
	if strings.Contains(address, "kubernetes:///") {
		//iresolver.RegisterInCluster()
		opts = append(opts, grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`))
	}
	if creds == nil {
		opts = append(opts, grpc.WithInsecure())
		conn, err = grpc.Dial(address, opts...)
	} else {
		opts = append(opts, grpc.WithTransportCredentials(creds))
		conn, err = grpc.Dial(address, opts...)
	}
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func grpcConnectionWithMaxCallRecvMsgSize(address string, creds credentials.TransportCredentials, maxCallRecvMsgSize int) (*grpc.ClientConn, error) {
	var conn *grpc.ClientConn
	var err error
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxCallRecvMsgSize)))
	if strings.Contains(address, "dns:///") {
		resolver.Register(dns.NewBuilder())
		opts = append(opts, grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`))
	}
	if strings.Contains(address, "kubernetes:///") {
		//iresolver.RegisterInCluster()
		opts = append(opts, grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`))
	}
	if creds == nil {
		opts = append(opts, grpc.WithInsecure())
		conn, err = grpc.Dial(address, opts...)
	} else {
		opts = append(opts, grpc.WithTransportCredentials(creds))
		conn, err = grpc.Dial(address, opts...)
	}
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// EndpointFactory returns endpoint factory
func EndpointFactory(makeEndpoint func(*grpc.ClientConn, time.Duration, stdopentracing.Tracer, log.Logger) endpoint.Endpoint, creds credentials.TransportCredentials, timeout time.Duration, tracer stdopentracing.Tracer, logger log.Logger) sd.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {

		if instance == "" {
			return nil, nil, errors.New("Empty instance")
		}

		conn, err := grpcConnection(instance, creds)
		if err != nil {
			logger.Log("host", instance, ulog.LogError, err.Error())
			return nil, nil, err
		}
		endpoint := makeEndpoint(conn, timeout, tracer, logger)

		return endpoint, conn, nil
	}
}

func EndpointFactoryWithMaxCallRecvMsgSize(makeEndpoint func(*grpc.ClientConn, time.Duration, stdopentracing.Tracer, log.Logger) endpoint.Endpoint, creds credentials.TransportCredentials, timeout time.Duration, tracer stdopentracing.Tracer, logger log.Logger, maxCallRecvMsgSize int) sd.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {

		if instance == "" {
			return nil, nil, errors.New("Empty instance")
		}

		conn, err := grpcConnectionWithMaxCallRecvMsgSize(instance, creds, maxCallRecvMsgSize)
		if err != nil {
			logger.Log("host", instance, ulog.LogError, err.Error())
			return nil, nil, err
		}
		endpoint := makeEndpoint(conn, timeout, tracer, logger)

		return endpoint, conn, nil
	}
}

func GrpcConnection(address string, creds credentials.TransportCredentials, cb *gobreaker.CircuitBreaker) (*grpc.ClientConn, error) {
	var conn *grpc.ClientConn
	var err error
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithUnaryInterceptor(circuitBreakerClientInterceptor(cb)))
	if strings.Contains(address, "dns:///") {
		opts = append(opts, grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`))
	}
	if strings.Contains(address, "kubernetes:///") {
		//iresolver.RegisterInCluster()
		opts = append(opts, grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`))
	}
	if creds == nil {
		opts = append(opts, grpc.WithInsecure())
		conn, err = grpc.Dial(address, grpc.WithInsecure())
	} else {
		opts = append(opts, grpc.WithTransportCredentials(creds))
		conn, err = grpc.Dial(address, grpc.WithTransportCredentials(creds))
	}
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func circuitBreakerClientInterceptor(cb *gobreaker.CircuitBreaker) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		_, cbErr := cb.Execute(func() (interface{}, error) {
			err := invoker(ctx, method, req, reply, cc, opts...)
			if err != nil {
				return nil, err
			}

			return nil, nil

		})
		return cbErr
	}
}
