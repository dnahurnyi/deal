//
// Copyright 2019
//
// @author: Denys Nahurnyi
// @email:  dnahurnyi@gmail.com
// ---------------------------------------------------------------------------
package dataSvc

import (
	"context"
	"errors"
	"fmt"

	"github.com/DenysNahurnyi/deal/common/utils"
	"github.com/DenysNahurnyi/deal/pb/generated/pb"
	"github.com/go-kit/kit/log"
	"github.com/mongodb/mongo-go-driver/mongo"
)

type Service interface {
	CreateUser(ctx context.Context, user *pb.User) (string, error)
	GetUser(ctx context.Context, userId string) (*pb.User, error)
}

type service struct {
	envType     string
	mongoClient *mongo.Client
	table       *mongo.Collection
}

func NewService(logger log.Logger, mgc *mongo.Client) (Service, error) {
	table := mgc.Database("travel").Collection("users")
	authSvc := utils.CreateAuthSvcClient(logger)
	fmt.Println("authSvc: ", authSvc)
	fmt.Println("Now I will call auth")
	ctx := context.Background()
	loginResp, err := authSvc.Login(ctx, &pb.LoginReq{
		ReqHdr: &pb.ReqHdr{
			Tid: "Transaction ID",
		},
		Username: "pisatel",
		Password: "123467",
	})
	fmt.Println("loginResp, err: ", loginResp, err)
	return &service{
		envType:     "test",
		mongoClient: mgc,
		table:       table,
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

func (s *service) GetUser(ctx context.Context, userID string) (*pb.User, error) {
	// userId, err := grpcutils.GetUserIDFromJWT(ctx)

	return GetUserByIdDB(ctx, userID, s.table)
}
