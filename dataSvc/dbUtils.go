package dataSvc

import (
	"context"
	"errors"
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
	DealDocs  []string           `bson:"deal_docs"`
	Offerings []string           `bson:"offerings"`
	Accepted  []string           `bson:"accepted"`
}

// ParticipantDB is an object of participant that stores in the DB
type ParticipantDB struct {
	ID       string `bson:"id,omitempty"`
	Accepted bool   `bson:"accepted"`
}

// SideDB is an object of side that stores in the DB
type SideDB struct {
	Type         pb.SideType     `bson:"type"`
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

func (dd *DealDocumentDB) toMongoFormat() bson.D {
	es := []bson.E{}
	if len(dd.Pacts) > 0 {
		es = append(es, bson.E{Key: "pacts", Value: dd.Pacts})
	}
	if len(dd.FinalVersion) > 0 {
		es = append(es, bson.E{Key: "final_version", Value: dd.FinalVersion})
	}
	if dd.Judge.Type == pb.SideType_JUDGE {
		es = append(es, bson.E{Key: "judge", Value: dd.Judge})
	}
	return es
}

func CreateUserDB(ctx context.Context, user UserDB, table *mongo.Collection) (string, error) {
	res, err := table.InsertOne(ctx, user)
	if err != nil {
		fmt.Println("Error creating user in mongo: ", err)
	}
	return res.InsertedID.(primitive.ObjectID).Hex(), err
}

func GetUserByIdDB(ctx context.Context, userId string, table *mongo.Collection) (*UserDB, error) {
	userIDDB, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		fmt.Println("Error creating object id to get user: ", err)
		return nil, err
	}
	userDB := &UserDB{}

	err = table.FindOne(ctx, bson.D{{Key: "_id", Value: userIDDB}}).Decode(userDB)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, nil
		}
		fmt.Println("Error getting user from mongo: ", err)
		return nil, err
	}

	return userDB, nil
}

func GetUserByIdDBConvert(ctx context.Context, userID string, table *mongo.Collection) (*pb.User, error) {
	userDB, err := GetUserByIdDB(ctx, userID, table)
	if err != nil || userDB == nil {
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
func GetDealDocByIdDB(ctx context.Context, dealDocID string, table *mongo.Collection) (*DealDocumentDB, error) {
	dealDocIDDB, err := primitive.ObjectIDFromHex(dealDocID)
	if err != nil {
		fmt.Println("Error creating object id to get user: ", err)
		return nil, err
	}
	dealDocDB := &DealDocumentDB{}

	err = table.FindOne(ctx, bson.D{{Key: "_id", Value: dealDocIDDB}}).Decode(dealDocDB)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, nil
		}
		fmt.Println("Error getting deal document from mongo: ", err)
		return nil, err
	}
	return dealDocDB, err
}

func GetDealDocByIdDBConvert(ctx context.Context, dealDocID string, table *mongo.Collection) (*pb.DealDocument, error) {
	dealDocDB, err := GetDealDocByIdDB(ctx, dealDocID, table)
	if err != nil {
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
					Members: int64(len(pact.Blue.Participants)),
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

// AcceptDealDocDB finds deal doc `dealDocID` in DB, and updates `Accepted` status to true of user `userID`, user should be on `side` side
func AcceptDealDocDB(ctx context.Context, dealDocID, userID string, side pb.SideType, dealDocTable, userTable *mongo.Collection) error {
	dealDocIDDB, err := primitive.ObjectIDFromHex(dealDocID)
	if err != nil {
		fmt.Println("Error creating object id to get deal document: ", err)
		return err
	}

	user, err := GetUserByIdDB(ctx, userID, userTable)
	if err != nil || user == nil {
		err = fmt.Errorf("Failed to get user by %q id", userID)
		fmt.Println("[ERROR]: ", err.Error())
		return err
	}
	userAccepted, err := userAcceptDeal(user, dealDocID)
	fmt.Println("userAccepted: ", userAccepted)
	if err != nil {
		fmt.Println("[ERROR]: ", "Failed to accept deal for user "+userID, err.Error())
		return err
	}
	err = UpdateUserDB(ctx, userID, userAccepted, userTable)
	if err != nil {
		fmt.Println("[ERROR]: ", "Failed to update user "+userID, err.Error())
		return err
	}

	dealDoc, err := GetDealDocByIdDB(ctx, dealDocID, dealDocTable)
	if err != nil {
		fmt.Println("Failed to get deal document: ", err)
		return err
	}
	dealDocAccepted, err := acceptDealForSide(dealDoc, side, userID)
	if err != nil {
		fmt.Println("Failed to accept deal: ", err)
		return err
	}
	_, err = dealDocTable.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: dealDocIDDB}},
		bson.D{{"$set", dealDocAccepted.toMongoFormat()}},
	)
	if err != nil {
		fmt.Println("Error updating deal document in mongo: ", err)
	}
	return err
}

// OfferDealDocDB finds deal doc `dealDocID` in DB, and add user `userID` to the `side` side
func OfferDealDocDB(ctx context.Context, dealDocID, userID string, side pb.SideType, table *mongo.Collection) error {
	dealDocIDDB, err := primitive.ObjectIDFromHex(dealDocID)
	if err != nil {
		fmt.Println("Error creating object id to get deal document: ", err)
		return err
	}
	dealDoc, err := GetDealDocByIdDB(ctx, dealDocID, table)
	if err != nil {
		fmt.Println("Failed to get deal document: ", err)
		return err
	}
	fmt.Println("Before: ", dealDoc.Pacts[0])
	dealDoc, err = offerDealForSide(dealDoc, side, userID)
	fmt.Println("After: ", dealDoc.Pacts[0])
	if err != nil {
		fmt.Println("Failed to offer the deal: ", err)
		return err
	}
	_, err = table.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: dealDocIDDB}},
		bson.D{{"$set", dealDoc.toMongoFormat()}},
	)
	if err != nil {
		fmt.Println("Error updating deal document in mongo: ", err)
	}
	return err
}

func offerDealForSide(dealDoc *DealDocumentDB, side pb.SideType, userID string) (*DealDocumentDB, error) {
	var err error
	if side != pb.SideType_JUDGE {
		// Find pact
		var pact *PactDB
		for i, pactTmp := range dealDoc.Pacts {
			if pactTmp.Version == dealDoc.FinalVersion {
				pact = &dealDoc.Pacts[i]
			}
		}
		if pact == nil {
			err = errors.New("Deal document is invalid, can't find needed pact")
			fmt.Println("[ERROR]: ", err.Error())
			return nil, err
		}
		//Find side in pact
		var pactSide *SideDB
		switch side {
		case pb.SideType_RED:
			pactSide = &pact.Red
		case pb.SideType_BLUE:
			pactSide = &pact.Blue
		}
		if pactSide == nil {
			err = fmt.Errorf("Invalid side %q", side)
			fmt.Println("[ERROR]: ", err.Error())
			return nil, err
		}
		//Create new participant `userID` on `pactSide`
		fmt.Println("before dealDoc: ", dealDoc)
		pactSide.Participants = append(pactSide.Participants, ParticipantDB{
			ID:       userID,
			Accepted: false,
		})
		fmt.Println("after dealDoc: ", dealDoc)
		return dealDoc, nil
	} else {
		// For judge
		//Create judge on judge side
		dealDoc.Judge.Participants = append(dealDoc.Judge.Participants, ParticipantDB{
			ID:       userID,
			Accepted: false,
		})
		return dealDoc, nil
	}
}

func acceptDealForSide(dealDoc *DealDocumentDB, side pb.SideType, userID string) (*DealDocumentDB, error) {
	var err error
	if side != pb.SideType_JUDGE {
		// Find pact
		var pact *PactDB
		for _, pactTmp := range dealDoc.Pacts {
			if pactTmp.Version == dealDoc.FinalVersion {
				pact = &pactTmp
			}
		}
		if pact == nil {
			err = errors.New("Deal document is invalid, can't find needed pact")
			fmt.Println("[ERROR]: ", err.Error())
			return nil, err
		}
		//Find side in pact
		var pactSide *SideDB
		switch side {
		case pb.SideType_RED:
			pactSide = &pact.Red
		case pb.SideType_BLUE:
			pactSide = &pact.Blue
		}
		if pactSide == nil {
			err = fmt.Errorf("Invalid side %q", side)
			err = fmt.Errorf("Failed to find user %q on %q side", userID, side)
			fmt.Println("[ERROR]: ", err.Error())
			return nil, err
		}
		//Find participant in side and accept
		for i, participant := range pactSide.Participants {
			if participant.ID == userID {
				if participant.Accepted == true {
					err = fmt.Errorf("Failed to accept, user %q already accepted this deal", userID)
					fmt.Println("[ERROR]: ", err.Error())
					return nil, err
				}
				pactSide.Participants[i].Accepted = true
				return dealDoc, nil
			}
		}
		err = fmt.Errorf("Failed to find user %q on %q side", userID, side)
		fmt.Println("[ERROR]: ", err.Error())
		return nil, err
	} else {
		// For judge
		//Find judge in judge side and accept
		if len(dealDoc.Judge.Participants) == 0 {
			err = fmt.Errorf("No judge side exist in this deal document")
			fmt.Println("[ERROR]: ", err.Error())
			return nil, err
		}
		for i, judge := range dealDoc.Judge.Participants {
			if judge.ID == userID {
				if judge.Accepted == true {
					err = fmt.Errorf("Failed to accept, judge %q already accepted this deal", userID)
					fmt.Println("[ERROR]: ", err.Error())
					return nil, err
				}
				dealDoc.Judge.Participants[i].Accepted = true
				return dealDoc, nil
			}
		}
		err = fmt.Errorf("Failed to judge %q on %q side", userID, side)
		fmt.Println("[ERROR]: ", err.Error())
		return nil, err
	}
}

// CreateDealDocumentDB create deal dcoument in the DB
func CreateDealDocumentDB(ctx context.Context, dealDocument DealDocumentDB, table *mongo.Collection) (string, error) {
	res, err := table.InsertOne(ctx, dealDocument)
	if err != nil {
		fmt.Println("Error creating deal document in mongo: ", err)
	}
	return res.InsertedID.(primitive.ObjectID).Hex(), err
}

func userAcceptDeal(user *UserDB, dealID string) (*UserDB, error) {
	var err error
	if user == nil {
		err = fmt.Errorf("Invalid user in input")
		return nil, err
	}
	initOfferLen := len(user.Offerings)
	for i, offerID := range user.Offerings {
		if offerID == dealID {
			user.Offerings = append(user.Offerings[:i], user.Offerings[i+1:]...)
			break
		}
	}
	if initOfferLen == len(user.Offerings) {
		err = fmt.Errorf("Failed to find offer %q in user offers", dealID)
		return nil, err
	}
	user.Accepted = append(user.Accepted, dealID)
	return user, err
}
