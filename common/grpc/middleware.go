//
// Copyright 2019
//
// @author: Denys Nahurnyi
// @email:  dnahurnyi@gmail.com
// ---------------------------------------------------------------------------
package grpcutils

// import (
// 	"bytes"
// 	"context"
// 	"crypto/rsa"
// 	"encoding/base64"
// 	"encoding/binary"
// 	"fmt"
// 	"math/big"
// 	"os"
// 	"strings"
// 	"sync"

// 	"github.com/aws/aws-sdk-go/aws"
// 	"github.com/aws/aws-sdk-go/service/dynamodb"
// 	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
// 	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
// 	jwt "github.com/dgrijalva/jwt-go"
// 	"github.com/go-kit/kit/log"
// 	"github.com/go-kit/kit/transport/grpc"
// 	"github.com/kalaspuffar/base64url"
// 	"github.com/${replaceMeBack}/api/common/constants"
// 	${replaceMeBack}err "github.com/${replaceMeBack}/api/common/errors"
// 	pb "github.com/${replaceMeBack}/api/common/pb/generated"
// 	"github.com/${replaceMeBack}/api/common/utils/logs"
// 	"google.golang.org/grpc/metadata"
// 	"google.golang.org/grpc/transport"
// )

// const (
// 	GRPCCOOKIES             = "grpcgateway-cookie"
// 	COOKIESHEADER           = "cookie"
// 	AUTHHEADER              = "authorization"
// 	REQUEST                 = "request"
// 	BEARER                  = "jwt"
// 	GRPCAUTHORIZATIONHEADER = "grpcgateway-authorization"
// 	META                    = "MetaInformation"
// 	CognitoJWTAlg           = "RS256"
// 	ERRCTX                  = "ErrorContext"
// 	TENANTIDATTR            = "custom:tenant_id"
// 	usernameAttr            = "cognito:username"
// )

// type ErrorContext struct {
// 	Error error
// }

// var cachedUserNames = make(map[string]*pb.TenantInfo) //Key is username, value is tenantInfo
// var rwMutex sync.RWMutex                              //Protects cachedUsedNames

// // This is middleware for adding cookies from request to context
// // As we can store in context only one value, so I would like to use next structure
// // request
// // --authorization header: value
// // --cookies:
// // ----key_1: val_1
// // ...so on, so use this structure for next middlewares
// func ParseCookies() grpc.ServerRequestFunc {
// 	return func(ctx context.Context, md metadata.MD) context.Context {
// 		var meta pb.MetaInfo
// 		// capital "Key" is illegal in HTTP/2.
// 		cookies, ok := md[GRPCCOOKIES]
// 		if !ok {
// 			return ctx
// 		}
// 		cookiesMap := parseCookiesFromStr(cookies[0])
// 		if ctx.Value(META) != nil {
// 			meta = ctx.Value(META).(pb.MetaInfo)
// 		} else {
// 			meta = pb.MetaInfo{
// 				Request: &pb.HttpRequest{
// 					Headers: make(map[string]string),
// 					Cookies: make(map[string]string),
// 				},
// 				Token: &pb.Token{
// 					Content: make(map[string]string),
// 				},
// 			}
// 		}
// 		for k, v := range cookiesMap {
// 			meta.Request.Cookies[k] = v
// 		}

// 		return context.WithValue(ctx, META, meta)
// 	}
// }

// func extractTokenFromAuthHeader(val string) (token string, ok bool) {
// 	authHeaderParts := strings.Split(val, " ")
// 	if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != BEARER {
// 		return "", false
// 	}

// 	return authHeaderParts[1], true
// }

// func parseCookiesFromStr(cookieStr string) map[string]string {
// 	cookies := make(map[string]string)
// 	cookiesParts := strings.Split(cookieStr, "; ")
// 	for _, val := range cookiesParts {
// 		tmpKeyVal := strings.Split(val, "=")
// 		k, v := tmpKeyVal[0], tmpKeyVal[1]
// 		cookies[k] = v
// 	}
// 	return cookies
// }

// // TODO: change header string to headers []string
// func ParseHeader(headerName string) grpc.ServerRequestFunc {
// 	return func(ctx context.Context, md metadata.MD) context.Context {
// 		var meta pb.MetaInfo
// 		// capital "Key" is illegal in HTTP/2.
// 		headerStr, ok := md[headerName]
// 		if !ok {
// 			return ctx
// 		}
// 		header, ok := extractTokenFromAuthHeader(headerStr[0])
// 		if !ok {
// 			return ctx
// 		}
// 		if ctx.Value(META) != nil {
// 			meta = ctx.Value(META).(pb.MetaInfo)
// 		} else {
// 			meta = pb.MetaInfo{
// 				Request: &pb.HttpRequest{
// 					Headers: make(map[string]string),
// 					Cookies: make(map[string]string),
// 				},
// 				Token: &pb.Token{
// 					Content: make(map[string]string),
// 				},
// 			}
// 		}

// 		oldHeaders := meta.Request.Headers
// 		meta.Request.Headers = make(map[string]string)
// 		for k, v := range oldHeaders {
// 			meta.Request.Headers[k] = v
// 		}
// 		meta.Request.Headers[headerName] = header

// 		return context.WithValue(ctx, META, meta)
// 	}
// }

// // VerifyToken: not exposed middleware function to verify jwt token from context
// // In case of success returns another context fulfilled by tenant specific info, for now it is just tenant_id
// // In case of error returns ErrorContext with one property error that we check through type assertation in each function
// func VerifyToken(dbif dynamodbiface.DynamoDBAPI, logger log.Logger) grpc.ServerRequestFunc {
// 	return func(ctx context.Context, md metadata.MD) context.Context {
// 		// Pull input info from context
// 		tmpMeta, ok := ctx.Value(META).(pb.MetaInfo)
// 		if !ok {
// 			var errCtx = ErrorContext{
// 				Error: ${replaceMeBack}err.NewError(nil, "jwt token is not present"),
// 			}
// 			return context.WithValue(ctx, ERRCTX, errCtx)
// 		}
// 		request := tmpMeta.Request
// 		tokenString := request.Headers[GRPCAUTHORIZATIONHEADER]
// 		var tenant *pb.TenantInfo
// 		var err error
// 		// Structure of this func could seem weird. First param is token string itself.
// 		// Second param is function that allow us to do some processing before give `jwt.Parse` key to verify.
// 		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
// 			if CognitoJWTAlg != token.Header["alg"] {
// 				errMsg := fmt.Sprintf("Unexpected signing method: %s", token.Header["alg"])
// 				return nil, ${replaceMeBack}err.NewError(nil, errMsg)
// 			}
// 			// Pull tenant id form token payload
// 			usernameRaw, ok := token.Claims.(jwt.MapClaims)[usernameAttr]
// 			if !ok {
// 				return nil, ${replaceMeBack}err.NewError(nil, "`custom:username` attribute missing in the jwt token")
// 			}
// 			tenant, err = GetTenantInfoFromUsername(ctx, dbif, logger, usernameRaw.(string))
// 			if err != nil {
// 				return nil, ${replaceMeBack}err.NewError(err, "User not found")
// 			}
// 			jwtCreds := tenant.UserpoolJwtCredentials
// 			// Get kid, kid is id to get appropriate pair of `n` and `e` from our stored list of pairs `jwtCreds`
// 			kid := token.Header["kid"].(string)
// 			if jwtCreds[kid] == nil {
// 				return nil, ${replaceMeBack}err.NewError(nil, "Invalid tenantId: "+tenant.GetTenantId())
// 			}
// 			if len(jwtCreds[kid].GetN()) == 0 {
// 				return nil, ${replaceMeBack}err.NewError(nil, "Missing N part of JWT public key")
// 			}
// 			if len(jwtCreds[kid].GetE()) == 0 {
// 				return nil, ${replaceMeBack}err.NewError(nil, "Missing E part of JWT public key")
// 			}
// 			// Generate key from `n` and `e` and return that key
// 			structuredPubKey, err := createRSAPublicKey(jwtCreds[kid].GetN(), jwtCreds[kid].GetE())
// 			if err != nil {
// 				return nil, err
// 			}
// 			return structuredPubKey, nil
// 		})
// 		// Error handling
// 		if err != nil && err.Error() == "Token is expired" {
// 			var errCtx = ErrorContext{
// 				Error: ${replaceMeBack}err.NewError(nil, "Token is expired"),
// 			}
// 			return context.WithValue(ctx, ERRCTX, errCtx)
// 		}
// 		if err != nil {
// 			var errCtx = ErrorContext{
// 				Error: err,
// 			}
// 			return context.WithValue(ctx, ERRCTX, errCtx)
// 		}
// 		claims, ok := token.Claims.(jwt.MapClaims)
// 		if !ok || !token.Valid {
// 			var errCtx = ErrorContext{
// 				Error: ${replaceMeBack}err.NewError(nil, "Invalid token"),
// 			}
// 			return context.WithValue(ctx, ERRCTX, errCtx)
// 		}
// 		if err = claims.Valid(); err != nil {
// 			var errCtx = ErrorContext{
// 				Error: ${replaceMeBack}err.NewError(nil, "Token payload is not valid"),
// 			}
// 			return context.WithValue(ctx, ERRCTX, errCtx)
// 		}
// 		// For backward compatibility
// 		// TBD: delete TENANTIDATTR
// 		claims[TENANTIDATTR] = tenant.GetTenantId()
// 		var tokenRes = &pb.Token{
// 			Content: make(map[string]string),
// 		}
// 		for k, v := range claims {
// 			tokenRes.Content[k] = fmt.Sprintf("%v", v)
// 		}
// 		var meta = pb.MetaInfo{
// 			Request: request,
// 			Token:   tokenRes,
// 		}

// 		return context.WithValue(ctx, META, meta)
// 	}
// }

// func createRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
// 	decN, err := base64url.Decode(nStr)
// 	if err != nil {
// 		return nil, err
// 	}
// 	n := big.NewInt(0)
// 	n.SetBytes(decN)

// 	decE, err := base64.StdEncoding.DecodeString(eStr)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var eBytes []byte
// 	if len(decE) < 8 {
// 		eBytes = make([]byte, 8-len(decE), 8)
// 		eBytes = append(eBytes, decE...)
// 	} else {
// 		eBytes = decE
// 	}
// 	eReader := bytes.NewReader(eBytes)
// 	var e uint64
// 	err = binary.Read(eReader, binary.BigEndian, &e)
// 	if err != nil {
// 		return nil, err
// 	}
// 	pubKey := rsa.PublicKey{N: n, E: int(e)}
// 	return &pubKey, nil

// }

// // GetJWTCreds : return jwt creds for given tenant_id, in case where tenant doesn't exist it returns ERRCODEENUM_DOES_NOT_EXIST error
// func GetJWTCreds(ctx context.Context, dbif dynamodbiface.DynamoDBAPI, logger log.Logger, tenantId string) (tenant *pb.TenantInfo, err error) {
// 	params := &dynamodb.GetItemInput{
// 		TableName: aws.String(constants.TENANTS_INFO_TABLE_NAME),
// 		Key: map[string]*dynamodb.AttributeValue{
// 			constants.NAMESPACE_KEY_NAME: {
// 				S: aws.String(GetNameSpace()),
// 			},
// 			constants.TENANT_INFO_KEY_NAME: {
// 				S: aws.String(tenantId),
// 			},
// 		},
// 		ProjectionExpression: aws.String("userpool_jwt_credentials, tenant_id"),
// 	}
// 	result, err := dbif.GetItemWithContext(ctx, params)
// 	if err != nil {
// 		return nil, ${replaceMeBack}err.NewError(err, "Failed to get tenant: "+tenantId)
// 	}
// 	err = dynamodbattribute.UnmarshalMap(result.Item, &tenant)
// 	if err != nil {
// 		return nil, ${replaceMeBack}err.NewError(err, "Failed to unmarshal tenant: "+tenantId)
// 	}
// 	if len(tenant.GetTenantId()) == 0 {
// 		return nil, ${replaceMeBack}err.NewError(nil, "Tenant, with id: "+tenantId+" does not exist")
// 	}
// 	return tenant, nil
// }

// // GetContextTenantFromJWT function to reduce repetative code for closed EP
// func GetContextTenantFromJWT(ctx context.Context) (string, error) {
// 	tenantID, err := getTokenKeyFromContext(ctx, TENANTIDATTR)
// 	if err != nil {
// 		return "", ${replaceMeBack}err.NewError(nil, "Failed to get tenant from token")
// 	}
// 	return tenantID, nil
// }

// func getTokenKeyFromContext(ctx context.Context, key string) (string, error) {
// 	if errCtx, ok := ctx.Value(ERRCTX).(ErrorContext); ok {
// 		return "", errCtx.Error
// 	}
// 	meta := ctx.Value(META).(pb.MetaInfo)
// 	value, ok := meta.Token.Content[key]
// 	if !ok {
// 		return "", ${replaceMeBack}err.NewError(nil, fmt.Sprintf("Failed to get [%s] from token", key))
// 	}
// 	return value, nil
// }

// // AuditLogEndpoint function to log endpoint invocation
// func AuditLogEndpoint(ctx context.Context, logger log.Logger, req string) {
// 	tenantID, _ := GetContextTenantFromJWT(ctx)
// 	username, _ := getTokenKeyFromContext(ctx, usernameAttr)
// 	stream, ok := transport.StreamFromContext(ctx)
// 	methodName := logs.UNDEFINED
// 	if ok {
// 		methodName = stream.Method()
// 	}
// 	${replaceMeBack}Logger := logs.New${replaceMeBack}Logger(logger)
// 	${replaceMeBack}Logger.AuditLog(methodName, username, tenantID, req)
// }

// // GetNameSpace uses the values of the environment variable to set a namespace
// func GetNameSpace() string {
// 	var nameSpace string

// 	nameSpaceName, nok := os.LookupEnv(constants.NAMESPACE_ENV_NAME)
// 	if nok && len(nameSpaceName) > 0 {
// 		nameSpace = nameSpaceName
// 	} else {
// 		panic("Missing Namespace configuration!")
// 	}

// 	return nameSpace
// }

// func IsProdBuild() bool {
// 	buildType, ok := os.LookupEnv(constants.ENV_BUILD_TYPE)
// 	if !ok || len(buildType) == 0 {
// 		return false
// 	}
// 	return buildType == constants.ENV_BUILD_TYPE_PROD
// }

// // GetTenantInfoFromUsername : return tenantInfo based on user email
// func GetTenantInfoFromUsername(ctx context.Context, dbif dynamodbiface.DynamoDBAPI, logger log.Logger, email string) (tenant *pb.TenantInfo, err error) {
// 	rwMutex.RLock()
// 	if _, ok := cachedUserNames[email]; ok {
// 		tenant = cachedUserNames[email]
// 		rwMutex.RUnlock()
// 		return tenant, nil
// 	}
// 	rwMutex.RUnlock()
// 	params := &dynamodb.GetItemInput{
// 		TableName: aws.String(constants.TENANT_USERS_TABLE_NAME),
// 		Key: map[string]*dynamodb.AttributeValue{
// 			constants.NAMESPACE_KEY_NAME: {
// 				S: aws.String(GetNameSpace()),
// 			},
// 			constants.EMAIL: {
// 				S: aws.String(email),
// 			},
// 		},
// 		ProjectionExpression: aws.String("tenant_id"),
// 	}
// 	result, err := dbif.GetItemWithContext(ctx, params)
// 	if err != nil {
// 		return nil, ${replaceMeBack}err.NewError(err, "Failed to get user: "+email)
// 	}
// 	var userInfo *pb.UserInfo
// 	err = dynamodbattribute.UnmarshalMap(result.Item, &userInfo)
// 	if err != nil {
// 		return nil, ${replaceMeBack}err.NewError(err, "Failed to unmarshal user: "+email)
// 	}
// 	if len(userInfo.GetTenantId()) == 0 {
// 		return nil, ${replaceMeBack}err.NewError(nil, "User "+email+"does not have any tenant")
// 	}
// 	tenant, err = GetJWTCreds(ctx, dbif, logger, userInfo.GetTenantId())
// 	if err != nil {
// 		return nil, ${replaceMeBack}err.NewError(err, "Tenant not found")
// 	}
// 	rwMutex.Lock()
// 	cachedUserNames[email] = tenant
// 	rwMutex.Unlock()
// 	return tenant, err
// }
