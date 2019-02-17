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

// ParticipantDB is an object of participant that stores in the DB
type ParticipantDB struct {
	ID       string
	Accepted bool
}

// SideDB is an object of side that stores in the DB
type SideDB struct {
	Type         pb.SideType
	Participants []ParticipantDB
}

// PactDB is an object of pact that stores in the DB
type PactDB struct {
	Content string
	Red     SideDB
	Blue    SideDB
	Timeout string
	Version string
}

// DealDocumentDB is an object of dealDocument(set of pacts) that stores in the DB
type DealDocumentDB struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Pacts        []PactDB
	FinalVersion string
	Judge        SideDB
}

func (u *UserDB) toMongoFormat() bson.D {
	es := []bson.E{}
	if len(u.Name) > 0 {
		es = append(es, bson.E{Key: "name", Value: u.Name})
	}
	if len(u.Surname) > 0 {
		es = append(es, bson.E{Key: "surname", Value: u.Surname})
	}
	if len(u.Username) > 0 {
		es = append(es, bson.E{Key: "username", Value: u.Username})
	}
	return es
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

func DeleteUserByIdDB(ctx context.Context, userId string, table *mongo.Collection) (*pb.User, error) {
	userIDDB, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		fmt.Println("Error creating object id to get user: ", err)
		return nil, err
	}
	userDB := UserDB{}

	err = table.FindOneAndDelete(ctx, bson.D{{Key: "_id", Value: userIDDB}}).Decode(&userDB)
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

// UpdateUserDB updates user in DB using userID to find it and user to update data
func UpdateUserDB(ctx context.Context, userID string, user *UserDB, table *mongo.Collection) error {
	userIDDB, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		fmt.Println("Error creating object id to get user: ", err)
		return err
	}
	_, err = table.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: userIDDB}},
		bson.D{{"$set", user.toMongoFormat()}},
	)
	if err != nil {
		fmt.Println("Error updating user in mongo: ", err)
	}
	return err
}

// CreateDealDocumentDB create deal dcoument in the DB
func CreateDealDocumentDB(ctx context.Context, dealDocument DealDocumentDB, table *mongo.Collection) (string, error) {
	res, err := table.InsertOne(ctx, dealDocument)
	if err != nil {
		fmt.Println("Error creating deal document in mongo: ", err)
	}
	return res.InsertedID.(primitive.ObjectID).Hex(), err
}
