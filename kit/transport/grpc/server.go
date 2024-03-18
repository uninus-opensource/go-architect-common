package grpc

import (
	logs "log"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	oldcontext "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// StreamServer ..
type StreamServer struct {
	e         endpoint.Endpoint
	before    []kitgrpc.ServerRequestFunc
	after     []kitgrpc.ServerResponseFunc
	finalizer []kitgrpc.ServerFinalizerFunc
	logger    log.Logger
}

// // BidirectionalStream ..
// type BidirectionalStream struct {
// 	ctx  ContextFunc
// 	dec  DecodeRequestStreamFunc
// 	enc  EncodeResponseStreamFunc
// 	recv RecvFunc
// 	send SendFunc
// }

// StreamHandler ..
type StreamHandler interface {
	ServeGRPCStream(oldcontext.Context, interface{}) (oldcontext.Context, interface{}, error)
}

// ServerOption sets an optional parameter for servers.
type ServerOption func(*StreamServer)

// ServerBefore functions are executed on the HTTP request object before the
// request is decoded.
func ServerBefore(before ...kitgrpc.ServerRequestFunc) ServerOption {
	return func(s *StreamServer) { s.before = append(s.before, before...) }
}

// ServerAfter functions are executed on the HTTP response writer after the
// endpoint is invoked, but before anything is written to the client.
func ServerAfter(after ...kitgrpc.ServerResponseFunc) ServerOption {
	return func(s *StreamServer) { s.after = append(s.after, after...) }
}

// ServerErrorLogger is used to log non-terminal errors. By default, no errors
// are logged.
func ServerErrorLogger(logger log.Logger) ServerOption {
	return func(s *StreamServer) { s.logger = logger }
}

// ServerFinalizer is executed at the end of every gRPC request.
// By default, no finalizer is registered.
func ServerFinalizer(f ...kitgrpc.ServerFinalizerFunc) ServerOption {
	return func(s *StreamServer) { s.finalizer = append(s.finalizer, f...) }
}

// NewStreamServer ..
func NewStreamServer(
	e endpoint.Endpoint,
	options ...ServerOption,
) *StreamServer {
	ss := &StreamServer{
		e:      e,
		logger: log.NewNopLogger(),
	}
	for _, option := range options {
		option(ss)
	}
	return ss
}

// // NewBidirectional ..
// func NewBidirectional(
// 	ctx ContextFunc,
// 	recv RecvFunc,
// 	send SendFunc,
// ) *BidirectionalStream {
// 	return &BidirectionalStream{
// 		ctx:  ctx,
// 		recv: recv,
// 		send: send,
// 	}
// }

// // InterfaceBidi ..
// type InterfaceBidi interface {
// 	Context() interface{}
// 	Send(interface{}) error
// 	Recv() (interface{}, error)
// }

// // Context ..
// func (bs *BidirectionalStream) Context() interface{} {
// 	return bs.ctx()
// }

// // Send ..
// func (bs *BidirectionalStream) Send(x interface{}) error {
// 	return bs.send(x)
// }

// // Recv ..
// func (bs *BidirectionalStream) Recv() (interface{}, error) {
// 	return bs.recv()
// }

// ServeGRPCStream ..
func (s StreamServer) ServeGRPCStream(ctx oldcontext.Context, param interface{}) (oldctx oldcontext.Context, resp interface{}, err error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}

	if len(s.finalizer) > 0 {
		defer func() {
			for _, f := range s.finalizer {
				f(ctx, err)
			}
		}()
	}

	for _, f := range s.before {
		ctx = f(ctx, md)
	}

	_, err = s.e(ctx, param)
	if err != nil {
		logs.Println(err)
		return ctx, nil, err
	}

	var mdHeader, mdTrailer metadata.MD
	for _, f := range s.after {
		ctx = f(ctx, &mdHeader, &mdTrailer)
	}

	if len(mdHeader) > 0 {
		if err = grpc.SendHeader(ctx, mdHeader); err != nil {
			s.logger.Log("err", err)
			return ctx, nil, err
		}
	}

	if len(mdTrailer) > 0 {
		if err = grpc.SetTrailer(ctx, mdTrailer); err != nil {
			s.logger.Log("err", err)
			return ctx, nil, err
		}
	}
	return ctx, nil, err
}

// func (s StreamServer) ServeGRPCStream(req interface{}) (resp interface{}, err error) {
// 	r := req.(*BidirectionalStream)
// 	var cek bool

// 	md, ok := metadata.FromIncomingContext(r.ctx)
// 	if !ok {
// 		md = metadata.MD{}
// 	}

// 	if len(s.finalizer) > 0 {
// 		defer func() {
// 			for _, f := range s.finalizer {
// 				f(r.ctx, err)
// 			}
// 		}()
// 	}

// 	for _, f := range s.before {
// 		r.ctx = f(r.ctx, md)
// 	}

// 	for {
// 		select {
// 		case <-r.ctx.Done():
// 			return nil, nil
// 		default:
// 		}

// 		receiver, err := r.recv()
// 		fmt.Println("=> recv:", receiver)
// 		if err == io.EOF {
// 			return nil, err
// 		}
// 		if err != nil {
// 			logs.Println(err)
// 			return nil, err
// 		}

// 		if !cek {

// 			request, err := r.dec(r.ctx, receiver)
// 			if err != nil {
// 				logs.Println(err)
// 				return nil, err
// 			}

// 			response, err := s.e(r.ctx, request)
// 			if err != nil {
// 				fmt.Println(err)
// 				return nil, err
// 			}

// 			var mdHeader, mdTrailer metadata.MD
// 			for _, f := range s.after {
// 				r.ctx = f(r.ctx, &mdHeader, &mdTrailer)
// 			}

// 			grpcStream, err := r.enc(r.ctx, response)
// 			if err != nil {
// 				logs.Println(err)
// 				return nil, err
// 			}
// 			r.send(grpcStream)

// 			var ep = func(ctx context.Context, val interface{}) (interface{}, error) {
// 				var ab []byte
// 				ab = val.([]byte)

// 				ay, err := r.pu(ab)
// 				if err != nil {
// 					logs.Println(err)
// 					return nil, err
// 				}

// 				request, err := r.dec(r.ctx, ay)
// 				if err != nil {
// 					logs.Println(err)
// 					return nil, err
// 				}

// 				response, err := s.e(r.ctx, request)
// 				if err != nil {
// 					logs.Println(err)
// 					return nil, err
// 				}

// 				grpcStream, err := r.enc(r.ctx, response)
// 				if err != nil {
// 					logs.Println(err)
// 					return nil, err
// 				}
// 				r.send(grpcStream)

// 				return nil, nil
// 			}
// 			timeoutConsum, err := r.mq(ep)
// 			if err != nil {
// 				logs.Println(err)
// 				return nil, err
// 			}

// 			defer r.mquns(timeoutConsum)
// 			cek = true
// 		}
// 	}
// }
