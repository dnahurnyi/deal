//
// Copyright 2019
//
// @author: Denys Nahurnyi
// @email:  dnahurnyi@gmail.com
// ---------------------------------------------------------------------------
package authSvc

import (
	"context"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson/primitive"

	"github.com/DenysNahurnyi/deal/pb/generated/pb"
	"github.com/go-kit/kit/log"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
)

type Service interface {
	Login(ctx context.Context, username, password string) (bool, error)
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

func (s *service) Login(ctx context.Context, username, password string) (bool, error) {
	fmt.Println("Your username:", username)
	fmt.Println("Your password", password)
	token, err := createToken(username)
	if err != nil {
		fmt.Println("Error creating token:", err)
	}
	fmt.Println("Created token:", token)
	collection := s.mongoClient.Database("testing").Collection("users")
	user := pb.User{}
	obId, err := primitive.ObjectIDFromHex("5c43766218b7b643868fab0f")
	fmt.Println("obId, err: ", obId, err)
	if err != nil {
		return false, err
	}
	// primitive.ObjectID
	err = collection.FindOne(ctx, bson.D{{Key: "_id", Value: obId}}).Decode(&user)
	fmt.Println("user:", user)
	// collection.FindId(bson.ObjectIdHex("5a2a75f777e864d018131a59")).One(&job)
	// err := s.mongoClient.Ping(ctx, readpref.Primary())
	// if err != nil {
	// 	fmt.Println("Error connecting to the DB: ", err)
	// 	return false, err
	// }
	// collection := s.mongoClient.Database("testing").Collection("users")
	// res, err := collection.InsertOne(ctx, *user)
	return true, nil
}
