//
// Copyright 2019
//
// @author: Denys Nahurnyi
// @email:  dnahurnyi@gmail.com
// ---------------------------------------------------------------------------
package authSvc

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/DenysNahurnyi/deal/pb/generated/pb"

	"github.com/go-kit/kit/log"
	"github.com/mongodb/mongo-go-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service interface {
	Login(ctx context.Context, username, password string) (string, error)
	SignUp(ctx context.Context, userReq *pb.CreateUserReq, password string) (string, error)
	GetKey(ctx context.Context) (string, int64, error)
	DeleteUser(ctx context.Context, tokenId string) error
	GetPubKey() *rsa.PublicKey
}

type service struct {
	envType       string
	mongoClient   *mongo.Client
	uKey          *rsa.PublicKey
	rKey          *rsa.PrivateKey
	table         *mongo.Collection
	dataSvcClient pb.DataServiceClient
}

func NewService(logger log.Logger, mgc *mongo.Client, dataSvcClient *pb.DataServiceClient) (Service, error) {
	rKey, uKey, err := loadKeys(false)
	if err != nil {
		fmt.Println("Failed to get keys")
		return nil, err
	}
	collection := mgc.Database("travel").Collection("usersSecure")

	return &service{
		envType:       "test",
		mongoClient:   mgc,
		uKey:          uKey,
		rKey:          rKey,
		table:         collection,
		dataSvcClient: *dataSvcClient,
	}, nil
}

func (s *service) GetPubKey() *rsa.PublicKey {
	return s.uKey
}

func (s *service) SignUp(ctx context.Context, userReq *pb.CreateUserReq, password string) (string, error) {
	if len(password) < 3 || len(password) > 30 {
		return "", status.Errorf(codes.InvalidArgument, "Validation error: password length must be in [3, 30] range")
	}

	if userReq == nil || userReq.GetUser() == nil {
		return "", status.Errorf(codes.InvalidArgument, "User object mustn't be empty")
	}
	if len(userReq.GetUser().GetUsername()) == 0 {
		return "", status.Errorf(codes.InvalidArgument, "Validation error: username length mustn't be empty")
	}
	userReq.User.Username = strings.TrimSpace(userReq.User.Username)

	user := UserDB{
		Username: userReq.GetUser().GetUsername(),
		Password: password,
	}

	// Check if this user exist in secure DB
	userGet, err := GetUserByUsernameDB(ctx, user.Username, s.table)
	if err != nil {
		fmt.Println("Failed to get user from DB")
		return "", err
	}
	if len(userGet.GetUsername()) > 0 {
		fmt.Println("[WARNING] user already exist")
		return "", errors.New("User already exist")
	}

	// Create user in common DB
	createUserDataRes, err := s.dataSvcClient.CreateUser(ctx, userReq)
	if err != nil {
		fmt.Println("Failed to create user in dataSvc: ", err)
		return "", err
	}
	user.TokenId = createUserDataRes.UserId

	// Create user in secure table
	userID, err := CreateUserDB(ctx, &user, s.table)
	if err != nil {
		fmt.Println("Failed to create user in dataSvc: ", err)
		// Call dataSvc to delete user
		return "", err
	}
	return userID, nil
}

func (s *service) Login(ctx context.Context, username, password string) (string, error) {
	if len(username) == 0 {
		return "", status.Errorf(codes.InvalidArgument, "Invalid params, username musn't be empty")
	}
	if len(password) == 0 {
		return "", status.Errorf(codes.InvalidArgument, "Invalid params, password musn't be empty")
	}
	userGet, err := GetUserByUsernameDB(ctx, username, s.table)
	if err != nil {
		return "", status.Errorf(codes.Internal, "Failed to get user from DB: %q", err)
	}
	if len(userGet.GetId()) == 0 {
		return "", status.Errorf(codes.InvalidArgument, "User does not exist")
	}

	jwtToken, err := createToken(userGet.GetId(), s.rKey)
	if err != nil {
		return "", status.Errorf(codes.Internal, "Error with token")
	}
	return jwtToken, nil
}

func (s *service) GetKey(ctx context.Context) (string, int64, error) {
	uKey := s.uKey
	if uKey == nil {
		msg := "Public key is not present in Auth service"
		return "", 0, errors.New(msg)
	}
	e := uKey.E
	nBytes := uKey.N.Bytes()
	if len(nBytes) == 0 {
		msg := "Bytes of public key is not valid"
		return "", 0, errors.New(msg)
	}
	return base64.StdEncoding.EncodeToString(nBytes), int64(e), nil
}

func (s *service) DeleteUser(ctx context.Context, tokenId string) error {
	return DeleteUserByTokenIdDB(ctx, tokenId, s.table)
}
