//
// Copyright 2019
//
// @author: Denys Nahurnyi
// @email:  dnahurnyi@gmail.com
// ---------------------------------------------------------------------------
package dataSvc

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"

	grpcutils "github.com/DenysNahurnyi/deal/common/grpc"
	"github.com/DenysNahurnyi/deal/pb/generated/pb"
	"github.com/go-kit/kit/log"
	"github.com/mongodb/mongo-go-driver/mongo"
)

type Service interface {
	CreateUser(ctx context.Context, user *pb.User) (string, error)
	GetUser(ctx context.Context) (*pb.User, error)
	DeleteUser(ctx context.Context) (*pb.User, error)
	GetPubKey() *rsa.PublicKey
}

type service struct {
	envType       string
	mongoClient   *mongo.Client
	table         *mongo.Collection
	authSvcClient pb.AuthServiceClient
	uKey          *rsa.PublicKey
}

func NewService(logger log.Logger, mgc *mongo.Client, authSvcClient *pb.AuthServiceClient) (Service, error) {
	table := mgc.Database("travel").Collection("users")
	ctx := context.Background()
	authSvcClientValue := *authSvcClient
	getPubKeyResp, err := authSvcClientValue.GetCheckTokenKey(ctx, &pb.EmptyReq{
		ReqHdr: &pb.ReqHdr{
			Tid: "call to get pub key",
		},
	})
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get pub key from authSvc, err:", err)
		return nil, err
	}
	uKey, err := grpcutils.CreatePubKey(getPubKeyResp.GetNBase64(), int(getPubKeyResp.GetE()))
	if err != nil {
		fmt.Println("[LOG]:", "Failed to create pub key for authSvc tokens, err:", err)
		return nil, err
	}
	fmt.Println("Data svc uKey: ", uKey)
	return &service{
		envType:       "test",
		mongoClient:   mgc,
		table:         table,
		authSvcClient: authSvcClientValue,
		uKey:          uKey,
	}, nil
}

func (s *service) CreateUser(ctx context.Context, userReq *pb.User) (string, error) {
	userGet, err := GetUserByUsernameDB(ctx, userReq.GetUsername(), s.table)
	if err != nil {
		fmt.Println("Failed to get user from DB")
		return "", err
	}
	if len(userGet.GetUsername()) > 0 {
		fmt.Println("[WARNING] user already exist")
		return "", errors.New("User already exist")
	}

	userID, err := CreateUserDB(ctx, &UserDB{
		Name:     userReq.GetName(),
		Surname:  userReq.GetSurname(),
		Username: userReq.GetUsername(),
	}, s.table)
	return userID, err
}

func (s *service) GetUser(ctx context.Context) (*pb.User, error) {
	userID, err := grpcutils.GetUserIDFromJWT(ctx)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get user id from token, err: ", err)
		return nil, err
	}
	fmt.Println("User id from JWT is: ", userID)

	return GetUserByIdDB(ctx, userID, s.table)
}

func (s *service) DeleteUser(ctx context.Context) (*pb.User, error) {
	userID, err := grpcutils.GetUserIDFromJWT(ctx)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get user id from token, err: ", err)
		return nil, err
	}

	user, err := DeleteUserByIdDB(ctx, userID, s.table)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to delete user, err: ", err)
		return nil, err
	}
	_, err = s.authSvcClient.DeleteUser(ctx, &pb.DeleteSecureUserReq{
		ReqHdr: &pb.ReqHdr{
			Tid: "some transaction id to delete user in auth",
		},
		TokenId: userID,
	})
	if err != nil {
		fmt.Println("[LOG]:", "Failed to delete user in auth service, err: ", err)
		return nil, err
	}
	return user, nil
}

func (s *service) GetPubKey() *rsa.PublicKey {
	return s.uKey
}
