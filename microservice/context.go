package microservice

import (
	"context"
	"time"

	"github.com/go-kit/kit/auth/jwt"
	"github.com/uninus-opensource/uninus-go-architect-common/uuid"
)

func GetContextUUID(ctx context.Context, ctxKey interface{}) uuid.UUID {
	val := ctx.Value(ctxKey)
	if val != nil {
		uid, ok := val.(uuid.UUID)
		if ok {
			return uid
		}

	}
	return uuid.Empty
}

// GetContextString return context value as type string
func GetContextString(ctx context.Context, ctxKey interface{}) string {
	val := ctx.Value(ctxKey)
	if val != nil {
		str, ok := val.(string)
		if ok {
			return str
		}
	}
	return ""
}

// GetContextFloat return context value as type string
func GetContextFloat(ctx context.Context, ctxKey interface{}) float64 {
	val := ctx.Value(ctxKey)
	if val != nil {
		flo, ok := val.(float64)
		if ok {
			return flo
		}
	}
	return 0
}

func GetContextTime(ctx context.Context, ctxKey interface{}) time.Time {
	val := ctx.Value(ctxKey)
	if val != nil {
		t, ok := val.(int64)
		if ok {
			time := time.Unix(t, 0)
			return time
		}
	}
	return time.Unix(0, 0)
}

// NewContextByContext is generate new context using context.
// this will get token from context and create new context
// this function will return new context
func NewContextByContext(ctx context.Context) context.Context {
	newCTX := context.WithValue(context.Background(), jwt.JWTTokenContextKey, GetTokenByContext(ctx))
	return newCTX
}

// GetTokenByContext .. TODO
func GetTokenByContext(ctx context.Context) string {
	return GetContextString(ctx, jwt.JWTTokenContextKey)
}

// SetRequestIDToContext .. TODO
func SetRequestIDToContext(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, CtxRequestID, requestID)
}

// GetRequestIDByContext .. TODO
func GetRequestIDByContext(ctx context.Context) string {
	return GetContextString(ctx, CtxRequestID)
}

func SetValueToContext(ctx context.Context, key contextKey, val interface{}) context.Context {
	return context.WithValue(ctx, key, val)
}

// CreateContext .. TODO
func CreateContext(token string) context.Context {
	return context.WithValue(context.Background(), jwt.JWTTokenContextKey, token)
}

// Spesific Request
func GetRequestContext(ctx context.Context) (requestUUID uuid.UUID, requestID, requestName string) {
	requestUUID = GetContextUUID(ctx, CtxRequestUUID)
	if requestUUID.IsEmpty() {
		requestUUID = GetContextUUID(ctx, CtxUserUUID)
	}
	requestID = GetContextString(ctx, CtxRequestID)
	requestName = GetContextString(ctx, CtxRequestUUID)
	if requestName == "" {
		requestName = GetContextString(ctx, CtxUserID)
	}
	return requestUUID, requestID, requestName
}

func SetRequestContext(ctx context.Context, requestUUID uuid.UUID, requestID, requestName string) context.Context {
	ctx = context.WithValue(ctx, CtxRequestUUID, requestUUID)
	ctx = context.WithValue(ctx, CtxRequestID, requestID)
	ctx = context.WithValue(ctx, CtxUserID, requestName)
	return ctx
}
