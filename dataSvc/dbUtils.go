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
	Name      string             `bson:"name,omitempty"`
	Surname   string             `bson:"surname,omitempty"`
	Username  string             `bson:"username,omitempty"`
	Id        primitive.ObjectID `bson:"_id,omitempty"`
	DealDocs  []string           `bson:"deal_docs,omitempty"`
	Offerings []string           `bson:"offerings,omitempty"`
}

// ParticipantDB is an object of participant that stores in the DB
type ParticipantDB struct {
	ID       string `bson:"id,omitempty"`
	Accepted bool   `bson:"accepted,omitempty"`
}

// SideDB is an object of side that stores in the DB
type SideDB struct {
	Type         pb.SideType     `bson:"type,omitempty"`
	Participants []ParticipantDB `bson:"participants,omitempty"`
}

// PactDB is an object of pact that stores in the DB
type PactDB struct {
	Content string `bson:"content,omitempty"`
	Red     SideDB `bson:"red,omitempty"`
	Blue    SideDB `bson:"blue,omitempty"`
	Timeout string `bson:"timeout,omitempty"`
	Version string `bson:"version,omitempty"`
}

// DealDocumentDB is an object of dealDocument(set of pacts) that stores in the DB
type DealDocumentDB struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Pacts        []PactDB           `bson:"pacts,omitempty"`
	FinalVersion string             `bson:"final_version,omitempty"`
	Judge        SideDB             `bson:"judge,omitempty"`
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
	if len(u.DealDocs) > 0 {
		es = append(es, bson.E{Key: "deal_docs", Value: u.DealDocs})
	}
	if len(u.Offerings) > 0 {
		es = append(es, bson.E{Key: "offerings", Value: u.Offerings})
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
		Name:      userDB.Name,
		Surname:   userDB.Surname,
		Username:  userDB.Username,
		DealDocs:  userDB.DealDocs,
		Offerings: userDB.Offerings,
	}
	return userRes, err
}

func GetDealDocByIdDB(ctx context.Context, dealDocID string, table *mongo.Collection) (*pb.DealDocument, error) {
	dealDocIDDB, err := primitive.ObjectIDFromHex(dealDocID)
	if err != nil {
		fmt.Println("Error creating object id to get user: ", err)
		return nil, err
	}
	dealDocDB := DealDocumentDB{}

	err = table.FindOne(ctx, bson.D{{Key: "_id", Value: dealDocIDDB}}).Decode(&dealDocDB)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, nil
		}
		fmt.Println("Error getting deal document from mongo: ", err)
		return nil, err
	}

	dealDocumentRes := &pb.DealDocument{
		Id:           dealDocDB.ID.Hex(),
		FinalVersion: dealDocDB.FinalVersion,
	}
	if len(dealDocDB.Judge.Participants) != 0 {
		participants := []*pb.Participant{}
		for _, judgeParticipant := range dealDocDB.Judge.Participants {
			partID := judgeParticipant.ID
			partAcceptance := judgeParticipant.Accepted
			participants = append(participants, &pb.Participant{
				Id:       partID,
				Accepted: partAcceptance,
			})
		}
		dealDocumentRes.Judge = &pb.Side{
			Members:      int64(len(participants)),
			Side:         pb.SideType_JUDGE,
			Participants: participants,
		}
	}

	if len(dealDocDB.Pacts) != 0 {
		dealDocumentRes.Pacts = make(map[string]*pb.Pact)
		for _, pact := range dealDocDB.Pacts {
			pactF := &pb.Pact{
				Content: pact.Content,
				Red: &pb.Side{
					Members: int64(len(pact.Red.Participants)),
					Side:    pb.SideType_RED,
				},
				Blue: &pb.Side{
					Members: int64(len(pact.Red.Participants)),
					Side:    pb.SideType_BLUE,
				},
				Timeout: pact.Timeout,
				Version: pact.Version,
			}
			for _, redParticipant := range pact.Red.Participants {
				pactF.Red.Participants = append(pactF.Red.Participants, &pb.Participant{
					Id:       redParticipant.ID,
					Accepted: redParticipant.Accepted,
				})
			}
			for _, blueParticipant := range pact.Blue.Participants {
				pactF.Blue.Participants = append(pactF.Blue.Participants, &pb.Participant{
					Id:       blueParticipant.ID,
					Accepted: blueParticipant.Accepted,
				})
			}
			dealDocumentRes.Pacts[pact.Version] = pactF
		}
	}
	return dealDocumentRes, err
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
		Name:      userDB.Name,
		Surname:   userDB.Surname,
		Username:  userDB.Username,
		DealDocs:  userDB.DealDocs,
		Offerings: userDB.Offerings,
	}
	return userRes, err
}

func GetUserByUsernameDB(ctx context.Context, username string, table *mongo.Collection) (*pb.User, string, error) {
	userDB := UserDB{}
	err := table.FindOne(ctx, bson.D{{Key: "username", Value: username}}).Decode(&userDB)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return &pb.User{}, "", nil
		}
		fmt.Println("Error getting user from mongo: ", err)
		return nil, "", err
	}

	userRes := &pb.User{
		Name:      userDB.Name,
		Surname:   userDB.Surname,
		Username:  userDB.Username,
		DealDocs:  userDB.DealDocs,
		Offerings: userDB.Offerings,
	}
	return userRes, userDB.Id.Hex(), err
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
