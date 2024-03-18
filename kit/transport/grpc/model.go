package grpc

const (
	// JWTTokenContextKey holds the key used to store a JWT Token in the
	// context.
	JWTTokenContextKey contextKey = "JWTToken"
	bearer             string     = "bearer"
)

type (
	// ContextFunc ..
	ContextFunc func() interface{}
	// RecvFunc ..
	RecvFunc func() (interface{}, error)
	// SendFunc ..
	SendFunc func(interface{}) error

	// contextKey ..
	contextKey string
)
