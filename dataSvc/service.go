//
// Copyright 2019
//
// @author: Denys Nahurnyi
// @email:  dnahurnyi@gmail.com
// ---------------------------------------------------------------------------
package dataSvc

import (
	"context"
	"fmt"

	"github.com/DenysNahurnyi/deal/pb/generated/pb"
	"github.com/go-kit/kit/log"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/readpref"
)

type Service interface {
	CreateUser(ctx context.Context, user *pb.User) (bool, error)
	GetUser(ctx context.Context, userId string) (*pb.User, error)
}

type service struct {
	envType     string
	mongoClient *mongo.Client
}

func NewService(logger log.Logger, mgc *mongo.Client) (Service, error) {
	return &service{
		envType:     "test",
		mongoClient: mgc,
	}, nil
}

func (s *service) CreateUser(ctx context.Context, user *pb.User) (bool, error) {
	fmt.Println("Your user is: ", *user)
	err := s.mongoClient.Ping(ctx, readpref.Primary())
	if err != nil {
		fmt.Println("Error connecting to the DB: ", err)
		return false, err
	}
	collection := s.mongoClient.Database("testing").Collection("users")
	res, err := collection.InsertOne(ctx, *user)
	id := res.InsertedID
	fmt.Println("Id of inserted: ", id)
	return true, nil
}

type UserDBStruct struct {
	name    string
	surname string
}

func (s *service) GetUser(ctx context.Context, userId string) (*pb.User, error) {
	err := s.mongoClient.Ping(ctx, readpref.Primary())
	if err != nil {
		fmt.Println("Error connecting to the DB: ", err)
		return nil, err
	}
	collection := s.mongoClient.Database("testing").Collection("users")
	user := pb.User{}
	err = collection.FindOne(ctx, bson.D{{Key: "name", Value: "Ben"}}).Decode(&user)
	if err != nil {
		fmt.Println("Error searching user: ", err)
	}
	fmt.Println("Searching user result: ", user)
	return nil, nil
}
