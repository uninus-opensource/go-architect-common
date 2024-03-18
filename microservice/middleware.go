package microservice

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/uninus-opensource/uninus-go-architect-common/uuid"
	kitjwt "github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/endpoint"
	jwt "github.com/golang-jwt/jwt/v4"

	"github.com/opentracing/opentracing-go"

	tlog "github.com/opentracing/opentracing-go/log"

)


// AuthenticateMiddleware adds function to validate token
func AuthenticateMiddleware(signKey []byte, signMethod string) endpoint.Middleware {
	signing := jwt.GetSigningMethod(signMethod)
	keyFunc := func(*jwt.Token) (interface{}, error) {
		var sign interface{}
		var err error
		switch signing {
		case jwt.SigningMethodES256, jwt.SigningMethodES384, jwt.SigningMethodES512:
			sign, err = jwt.ParseECPublicKeyFromPEM(signKey)
		case jwt.SigningMethodRS256, jwt.SigningMethodRS384, jwt.SigningMethodRS512,
			jwt.SigningMethodPS256, jwt.SigningMethodPS384, jwt.SigningMethodPS512:
			sign, err = jwt.ParseRSAPublicKeyFromPEM(signKey)
		default:
			sign = signKey
		}
		if err != nil {
			return nil, err
		}
		return sign, nil
	}

	return endpoint.Chain(kitjwt.NewParser(keyFunc, signing, kitjwt.MapClaimsFactory), userContextMiddleware())
}

// NoAuthenticateMiddleware adds function to validate token
func NoAuthenticateMiddleware() endpoint.Middleware {
	return endpoint.Chain(CustomParser(), userContextMiddleware())
}

func CustomParser() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			// tokenString is stored in the context from the transport handlers.
			tokenString, ok := ctx.Value(kitjwt.JWTTokenContextKey).(string)
			if !ok {
				return nil, errors.New("token up for parsing was not passed through the context")
			}

			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return nil, nil
			})

			ctx = context.WithValue(ctx, kitjwt.JWTClaimsContextKey, token.Claims)
			return next(ctx, request)
		}
	}
}

// AuthorizeMiddleware adds function to validate authorization
func AuthorizeMiddleware(serviceID, operationID int64) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			claims := ctx.Value(kitjwt.JWTClaimsContextKey)
			mapClaims := claims.(jwt.MapClaims)
			mapACL := mapClaims[CtxACL.String()]
			mapACLS := mapClaims[CtxACLS.String()]
			if mapACL == nil && mapACLS == nil {
				return nil, ErrUnauthorized
			}

			var acl map[string]interface{}
			if mapACLS != nil {
				acl = mapACLS.(map[string]interface{})
			} else {
				acl = mapACL.(map[string]interface{})
			}

			sid := strconv.Itoa(int(serviceID))
			access := acl[sid]
			if access == nil {
				fmt.Println("[TOKEN]", fmt.Sprintf("%+v", mapClaims))
				fmt.Println("[ERROR Unauthorized Service]", sid)
				return nil, ErrUnauthorized
			}

			var oaccess uint64

			if mapACLS != nil {
				oaccess, _ = strconv.ParseUint(access.(string), 10, 64)
			} else {
				oaccess = uint64(access.(float64))
			}

			var one uint64 = 1
			oid := one << uint64(operationID-1)
			if oid&oaccess == 0 {
				fmt.Println("[TOKEN]", fmt.Sprintf("%+v", mapClaims))
				fmt.Println("[ERROR Unauthorized Operation]", fmt.Sprintf("%d", operationID))
				fmt.Println("[ERROR Unauthorized Access]", fmt.Sprintf("%+v", access))
				return nil, ErrUnauthorized
			}

			return next(ctx, request)
		}
	}
}

// AuthorizeExistMiddleware adds function to validate authorization at least one
func AuthorizeExistMiddleware(serviceID int64, operationID []int64) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			claims := ctx.Value(kitjwt.JWTClaimsContextKey)
			mapClaims := claims.(jwt.MapClaims)
			mapACL := mapClaims[CtxACL.String()]
			mapACLS := mapClaims[CtxACLS.String()]
			if mapACL == nil && mapACLS == nil {
				return nil, ErrUnauthorized
			}

			var acl map[string]interface{}
			if mapACLS != nil {
				acl = mapACLS.(map[string]interface{})
			} else {
				acl = mapACL.(map[string]interface{})
			}

			sid := strconv.Itoa(int(serviceID))
			access, ok := acl[sid]
			if !ok {
				return nil, ErrUnauthorized
			}

			var oaccess uint64

			if mapACLS != nil {
				oaccess, _ = strconv.ParseUint(access.(string), 10, 64)
			} else {
				oaccess = uint64(access.(float64))
			}

			exist := false
			for _, opID := range operationID {
				var one uint64 = 1
				oid := one << uint64(opID-1)
				if oid&oaccess != 0 {
					exist = true
					break
				}
			}
			if !exist {
				return nil, ErrUnauthorized
			}

			return next(ctx, request)
		}
	}
}

func ClaimsMiddleware() endpoint.Middleware {
	middleware := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			strToken := GetTokenByContext(ctx)

			token, _, err := new(jwt.Parser).ParseUnverified(strToken, jwt.MapClaims{})
			if err != nil {
				return nil, kitjwt.ErrTokenMalformed
			}

			ctx = context.WithValue(ctx, kitjwt.JWTClaimsContextKey, token.Claims)
			return next(ctx, request)
		}
	}

	return endpoint.Chain(middleware, userContextMiddleware())
}

func ClaimsOrBasicMiddleware(apiKey string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			strToken := GetTokenByContext(ctx)

			if strToken == apiKey {
				return next(ctx, request)
			}

			token, _, err := new(jwt.Parser).ParseUnverified(strToken, jwt.MapClaims{})
			if err != nil {
				return nil, kitjwt.ErrTokenMalformed
			}

			ctx = context.WithValue(ctx, kitjwt.JWTClaimsContextKey, token.Claims)
			return userContextMiddleware()(next)(ctx, request)
		}
	}
}

// userContextMiddleware extracted user info from jwt claim
func userContextMiddleware() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			claims := ctx.Value(kitjwt.JWTClaimsContextKey)
			mapClaims := claims.(jwt.MapClaims)
			return next(getUserContext(ctx, mapClaims), request)
		}
	}
}

func getUserContext(ctx context.Context, claims jwt.MapClaims) context.Context {
	userContext := ctx
	uid, ok := claims[CtxUserUUID.String()]
	if ok {
		userID, fine := uid.(string)
		if fine {
			userUUID, err := uuid.FromString(userID)
			if err == nil {
				userContext = context.WithValue(userContext, CtxUserUUID, userUUID)
			}
		}
	} else {
		userContext = context.WithValue(userContext, CtxUserUUID, uuid.Empty)
	}

	user, ok := claims[CtxUserID.String()]
	if ok {
		userContext = context.WithValue(userContext, CtxUserID, user)
	}

	name, ok := claims[CtxUname.String()]
	if ok {
		userContext = context.WithValue(userContext, CtxUname, name)
	}

	domain, ok := claims[CtxDomain.String()]
	if ok {
		userContext = context.WithValue(userContext, CtxDomain, domain)
	}

	email, ok := claims[CtxEmail.String()]
	if ok {
		userContext = context.WithValue(userContext, CtxEmail, email)
	}

	phone, ok := claims[CtxPhone.String()]
	if ok {
		userContext = context.WithValue(userContext, CtxPhone, phone)
	}

	domainID, ok := claims[CtxDomainID.String()]
	if ok {
		domID, str := domainID.(string)
		if str {
			domUUID, err := uuid.FromString(domID)
			if err == nil {
				userContext = context.WithValue(userContext, CtxDomainID, domUUID)
			}
		} else {
			userContext = context.WithValue(userContext, CtxDomainID, uuid.Empty)
		}

	}

	domainName, ok := claims[CtxDomainName.String()]
	if ok {
		userContext = context.WithValue(userContext, CtxDomainName, domainName)
	}

	domainType, ok := claims[CtxDomainType.String()]
	if ok {
		userContext = context.WithValue(userContext, CtxDomainType, domainType)
	}

	expired, ok := claims[CtxExp.String()]
	if ok {
		userContext = context.WithValue(userContext, CtxExp, expired)
	}

	groupName, ok := claims[CtxGroupName.String()]
	if ok {
		userContext = context.WithValue(userContext, CtxGroupName, groupName)
	}

	return userContext
}

// SetTagSpan to set
type SetTagSpan func(span opentracing.Span, req interface{}) opentracing.Span

// TraceMiddleware is for tracing
func TraceMiddleware(tracer opentracing.Tracer, operationName string, setSpanRequest SetTagSpan, setSpanResponse SetTagSpan) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			span := opentracing.SpanFromContext(ctx)
			if span == nil {
				span = tracer.StartSpan(operationName)
			} else {
				span = tracer.StartSpan(operationName, opentracing.ChildOf(span.Context()))
			}
			defer span.Finish()

			if setSpanRequest != nil && request != nil {
				setSpanRequest(span, request)
			}

			ctx = opentracing.ContextWithSpan(ctx, span)

			success, err := next(ctx, request)

			if err != nil {
				span.LogFields(tlog.String("error", err.Error()))
			}

			if setSpanResponse != nil && success != nil {
				setSpanResponse(span, success)
			}

			return success, err

			//return next(ctx, request)
		}
	}
}

// SecurityPolicyMiddleware adds function to validate token
// func SecurityPolicyMiddleware() endpoint.Middleware {
// 	return func(next endpoint.Endpoint) endpoint.Endpoint {
// 		return func(ctx context.Context, request interface{}) (interface{}, error) {
// 			return next(getSecurityPolicy(ctx), request)
// 		}
// 	}
// }

// FormatingMsgErrorMiddleware is formating msg error
func FormatingMsgErrorMiddleware() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			// fmt.Printf("FormatingMsgErrorMiddleware : %+v - %v\n", ctx, request)
			response, err := next(ctx, request)
			if err != nil {
				// fmt.Printf("FormatingMsgErrorMiddleware : %+v - %v\n", response, err)
				return response, formatingMsgError(err)
			}
			return response, nil
		}
	}
}
