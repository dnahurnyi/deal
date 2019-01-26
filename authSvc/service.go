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
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson/primitive"

	"github.com/go-kit/kit/log"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
)

type Service interface {
	Login(ctx context.Context, username, password string) (string, error)
}

type service struct {
	envType     string
	mongoClient *mongo.Client
	uKey        *rsa.PublicKey
	rKey        *rsa.PrivateKey
	table       *mongo.Collection
}

func NewService(logger log.Logger, mgc *mongo.Client) (Service, error) {
	rKey, uKey, err := loadKeys(false)
	if err != nil {
		fmt.Println("Failed to get keys")
		return nil, err
	}
	collection := mgc.Database("testing").Collection("users")

	return &service{
		envType:     "test",
		mongoClient: mgc,
		uKey:        uKey,
		rKey:        rKey,
		table:       collection,
	}, nil
}

type UserDB struct {
	Name    string
	Surname string
	Id      primitive.ObjectID `bson:"_id,omitempty"`
}

func (user UserDB) GetUserID() string {
	return user.Id.Hex()
}

func (s *service) Login(ctx context.Context, username, password string) (string, error) {
	userDB := UserDB{}

	err := s.table.FindOne(ctx, bson.D{
		{Key: "username", Value: username},
		{Key: "password", Value: password},
	}).Decode(&userDB)
	if err != nil {
		return "", err
	}

	return createToken(userDB.GetUserID(), s.rKey)

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
