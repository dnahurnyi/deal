package grpcutils

import (
	"context"
	"errors"
	"fmt"
	"strings"

	pb "github.com/DenysNahurnyi/deal/pb/generated/pb"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	GRPCAUTHORIZATIONHEADER = "grpcgateway-authorization"
	MySecret                = "secretWordDorJWTSign"
	META                    = "MetaInformation"
	BEARER                  = "jwt"
	GRPCCOOKIES             = "grpcgateway-cookie"
	ERRCTX                  = "ErrorContext"
	USER_ID                 = "userId"
)

type ErrorContext struct {
	Error error
}

// VerifyToken: not exposed middleware function to verify jwt token from context
// In case of success returns another context fulfilled by tenant specific info, for now it is just tenant_id
// In case of error returns ErrorContext with one property error that we check through type assertation in each function
func VerifyToken() grpc.ServerRequestFunc {
	return func(ctx context.Context, md metadata.MD) context.Context {

		// Pull input info from context
		tmpMeta, ok := ctx.Value(META).(pb.MetaInfo)
		fmt.Println("IN VerifyToken: ", ok)

		if !ok {
			return ctx
		}
		request := tmpMeta.Request
		tokenString := request.Headers[GRPCAUTHORIZATIONHEADER]
		userID, err := checkToken(tokenString)
		fmt.Println("IN VerifyToken: ", userID, err)

		var tokenRes = &pb.Token{
			Content: map[string]string{
				"userId": userID,
			},
		}
		var meta = pb.MetaInfo{
			Request: request,
			Token:   tokenRes,
		}

		return context.WithValue(ctx, META, meta)
	}
}

func checkToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(MySecret), nil
	})

	if err != nil {
		fmt.Println("Error parsing token:", err)
		return "", err
	}

	// Error handling
	// if err != nil && err.Error() == "Token is expired" {
	// 	var errCtx = ErrorContext{
	// 		Error: ${replaceMeBack}err.NewError(nil, "Token is expired"),
	// 	}
	// 	return context.WithValue(ctx, ERRCTX, errCtx)
	// }
	// if err != nil {
	// 	var errCtx = ErrorContext{
	// 		Error: err,
	// 	}
	// 	return context.WithValue(ctx, ERRCTX, errCtx)
	// }
	// claims, ok := token.Claims.(jwt.MapClaims)
	// if !ok || !token.Valid {
	// 	var errCtx = ErrorContext{
	// 		Error: ${replaceMeBack}err.NewError(nil, "Invalid token"),
	// 	}
	// 	return context.WithValue(ctx, ERRCTX, errCtx)
	// }
	// if err = claims.Valid(); err != nil {
	// 	var errCtx = ErrorContext{
	// 		Error: ${replaceMeBack}err.NewError(nil, "Token payload is not valid"),
	// 	}
	// 	return context.WithValue(ctx, ERRCTX, errCtx)
	// }

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if userId, ok := claims["userId"]; ok {
			return userId.(string), nil
		}
	}
	return "", err
}

// This is middleware for adding cookies from request to context
// As we can store in context only one value, so I would like to use next structure
// request
// --authorization header: value
// --cookies:
// ----key_1: val_1
// ...so on, so use this structure for next middlewares
func ParseCookies() grpc.ServerRequestFunc {
	return func(ctx context.Context, md metadata.MD) context.Context {
		var meta pb.MetaInfo
		// capital "Key" is illegal in HTTP/2.
		cookies, ok := md[GRPCCOOKIES]
		if !ok {
			return ctx
		}
		cookiesMap := parseCookiesFromStr(cookies[0])
		if ctx.Value(META) != nil {
			meta = ctx.Value(META).(pb.MetaInfo)
		} else {
			meta = pb.MetaInfo{
				Request: &pb.HttpRequest{
					Headers: make(map[string]string),
					Cookies: make(map[string]string),
				},
				Token: &pb.Token{
					Content: make(map[string]string),
				},
			}
		}
		for k, v := range cookiesMap {
			meta.Request.Cookies[k] = v
		}

		return context.WithValue(ctx, META, meta)
	}
}

func extractTokenFromAuthHeader(val string) (token string, ok bool) {
	authHeaderParts := strings.Split(val, " ")
	if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != BEARER {
		return "", false
	}

	return authHeaderParts[1], true
}

func parseCookiesFromStr(cookieStr string) map[string]string {
	cookies := make(map[string]string)
	cookiesParts := strings.Split(cookieStr, "; ")
	for _, val := range cookiesParts {
		tmpKeyVal := strings.Split(val, "=")
		k, v := tmpKeyVal[0], tmpKeyVal[1]
		cookies[k] = v
	}
	return cookies
}

// TODO: change header string to headers []string
func ParseHeader(headerName string) grpc.ServerRequestFunc {
	return func(ctx context.Context, md metadata.MD) context.Context {
		var meta pb.MetaInfo
		// capital "Key" is illegal in HTTP/2.
		headerStr, ok := md[headerName]
		if !ok {
			return ctx
		}
		header, ok := extractTokenFromAuthHeader(headerStr[0])
		if !ok {
			return ctx
		}
		if ctx.Value(META) != nil {
			meta = ctx.Value(META).(pb.MetaInfo)
		} else {
			meta = pb.MetaInfo{
				Request: &pb.HttpRequest{
					Headers: make(map[string]string),
					Cookies: make(map[string]string),
				},
				Token: &pb.Token{
					Content: make(map[string]string),
				},
			}
		}

		oldHeaders := meta.Request.Headers
		meta.Request.Headers = make(map[string]string)
		for k, v := range oldHeaders {
			meta.Request.Headers[k] = v
		}
		meta.Request.Headers[headerName] = header

		return context.WithValue(ctx, META, meta)
	}
}

// GetContextTenantFromJWT function to reduce repetative code for closed EP
func GetUserIDFromJWT(ctx context.Context) (string, error) {
	tenantID, err := getTokenKeyFromContext(ctx, USER_ID)
	if err != nil {
		return "", errors.New("Failed to get tenant from token")
	}
	return tenantID, nil
}

func getTokenKeyFromContext(ctx context.Context, key string) (string, error) {
	if errCtx, ok := ctx.Value(ERRCTX).(ErrorContext); ok {
		return "", errCtx.Error
	}
	meta := ctx.Value(META).(pb.MetaInfo)
	value, ok := meta.Token.Content[key]
	if !ok {
		return "", errors.New(fmt.Sprintf("Failed to get [%s] from token", key))
	}
	return value, nil
}
