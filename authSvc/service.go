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
	"errors"
	"fmt"

	"github.com/DenysNahurnyi/deal/pb/generated/pb"

	"github.com/go-kit/kit/log"
	"github.com/mongodb/mongo-go-driver/mongo"
)

type Service interface {
	Login(ctx context.Context, username, password string) (string, error)
	SignUp(ctx context.Context, userReq *pb.CreateUserReq, password string) (string, error)
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
	fmt.Println("[travel DB connected]")

	return &service{
		envType:       "test",
		mongoClient:   mgc,
		uKey:          uKey,
		rKey:          rKey,
		table:         collection,
		dataSvcClient: *dataSvcClient,
	}, nil
}

func (s *service) SignUp(ctx context.Context, userReq *pb.CreateUserReq, password string) (string, error) {
	fmt.Println("[SignUp method called]")
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
	// userDB := UserDB{}

	// res, err := table.InsertOne(ctx, *user)
	// if err != nil {
	// 	fmt.Println("Error creating user in mongo: ", err)
	// }
	// return res.InsertedID.(primitive.ObjectID).Hex(), err

	// err := s.table.FindOne(ctx, bson.D{
	// 	{Key: "username", Value: username},
	// 	{Key: "password", Value: password},
	// }).Decode(&userDB)
	// if err != nil {
	// 	return "", err
	// }

	// return createToken(userDB.GetUserID(), s.rKey)
	return "in development", nil

	// token, err := createToken(userDB.Id.Hex())
	// if err != nil {
	// 	fmt.Println("Error creating token:", err)
	// 	return "", err
	// }
	// fmt.Println("#1")
	// err = createKeys("SuperSecret")
	// obId, err := primitive.ObjectIDFromHex("5c43766218b7b643868fab0f")
	// fmt.Println("obId, err: ", obId, err)
	// if err != nil {
	// 	return "", err
	// }
	// primitive.ObjectID
	// err = collection.FindOne(ctx, bson.D{{Key: "_id", Value: obId}}).Decode(&user)
	// fmt.Println("user:", user)
	// collection.FindId(bson.ObjectIdHex("5a2a75f777e864d018131a59")).One(&job)
	// err := s.mongoClient.Ping(ctx, readpref.Primary())
	// if err != nil {
	// 	fmt.Println("Error connecting to the DB: ", err)
	// 	return false, err
	// }
	// collection := s.mongoClient.Database("testing").Collection("users")
	// res, err := collection.InsertOne(ctx, *user)
}
