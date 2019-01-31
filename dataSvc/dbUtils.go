package dataSvc

import (
	"context"
	"fmt"

	pb "github.com/DenysNahurnyi/deal/pb/generated/pb"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/primitive"
	"github.com/mongodb/mongo-go-driver/mongo"
)

type UserDB struct {
	Name     string
	Surname  string
	Username string
	Id       primitive.ObjectID `bson:"_id,omitempty"`
}

func CreateUserDB(ctx context.Context, user *UserDB, table *mongo.Collection) (string, error) {
	res, err := table.InsertOne(ctx, *user)
	if err != nil {
		fmt.Println("Error creating user in mongo: ", err)
	}
	return res.InsertedID.(primitive.ObjectID).Hex(), err
}

func GetUserByIdDB(ctx context.Context, userId string, table *mongo.Collection) (*pb.User, error) {
	userIDDB, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		fmt.Println("Error creating object id to get user: ", err)
		return nil, err
	}
	userDB := UserDB{}

	err = table.FindOne(ctx, bson.D{{Key: "_id", Value: userIDDB}}).Decode(&userDB)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return &pb.User{}, nil
		}
		fmt.Println("Error getting user from mongo: ", err)
		return nil, err
	}

	userRes := &pb.User{
		Name:     userDB.Name,
		Surname:  userDB.Surname,
		Username: userDB.Username,
	}
	return userRes, err
}

func GetUserByUsernameDB(ctx context.Context, username string, table *mongo.Collection) (*pb.User, error) {
	userDB := UserDB{}
	err := table.FindOne(ctx, bson.D{{Key: "username", Value: username}}).Decode(&userDB)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return &pb.User{}, nil
		}
		fmt.Println("Error getting user from mongo: ", err)
		return nil, err
	}

	userRes := &pb.User{
		Name:     userDB.Name,
		Surname:  userDB.Surname,
		Username: userDB.Username,
	}
	return userRes, err
}
