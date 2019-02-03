package authSvc

import (
	"context"
	"fmt"

	pb "github.com/DenysNahurnyi/deal/pb/generated/pb"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/primitive"
	"github.com/mongodb/mongo-go-driver/mongo"
)

type UserDB struct {
	Username string
	Password string
	TokenId  string
	Id       primitive.ObjectID `bson:"_id,omitempty"`
}

func (user UserDB) GetUserID() string {
	return user.Id.Hex()
}

func CreateUserDB(ctx context.Context, user *UserDB, table *mongo.Collection) (string, error) {
	res, err := table.InsertOne(ctx, *user)
	if err != nil {
		fmt.Println("Error creating user in mongo: ", err)
	}
	return res.InsertedID.(primitive.ObjectID).Hex(), err
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
		Username: userDB.Username,
		Password: userDB.Password,
		Id:       userDB.TokenId,
	}
	return userRes, err
}

func DeleteUserByTokenIdDB(ctx context.Context, tokenID string, table *mongo.Collection) error {
	userDB := UserDB{}

	err := table.FindOneAndDelete(ctx, bson.D{{Key: "tokenid", Value: tokenID}}).Decode(&userDB)
	if err != nil {
		fmt.Println("Error getting user from mongo: ", err)
		return err
	}
	return nil
}
