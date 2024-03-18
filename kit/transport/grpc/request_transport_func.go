package grpc

import (
	"context"
)

// DecodeRequestStreamFunc ..
type DecodeRequestStreamFunc func(context.Context, interface{}) (interface{}, error)

// EncodeResponseStreamFunc ..
type EncodeResponseStreamFunc func(context.Context, interface{}) (interface{}, error)
