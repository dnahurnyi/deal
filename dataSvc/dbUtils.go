package dataSvc

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	pb "github.com/DenysNahurnyi/deal/pb/generated/pb"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/primitive"
	"github.com/mongodb/mongo-go-driver/mongo"
)

type UserDB struct {
	Name          string             `bson:"name,omitempty"`
	Surname       string             `bson:"surname,omitempty"`
	Username      string             `bson:"username,omitempty"`
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	DealDocs      []string           `bson:"deal_docs"`
	Offerings     []string           `bson:"offerings"`
	Accepted      []string           `bson:"accepted"`
	Participating []string           `bson:"participating"`
	DealResults   []string           `bson:"deal_results"`
	IsJudge       bool               `bson:"is_judge"`
	JudgeProfile  *JudgeProfile      `bson:"judge_profile"`
}

// JudgeProfile is an object judge profile the stores in the DB
type JudgeProfile struct {
	Propositions   []string   `bson:"propositions"`
	Participatings []string   `bson:"participatings"`
	Decisions      []Decision `bson:"decisions"`
}

// Decision is an object that contains info about judge decision
type Decision struct {
	DealID string `bson:"deal_id"`
	Winner string `bson:"winner"`
	When   string `bson:"when"`
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

// Status is an object of Status that stores in the DB
type Status struct {
	Name string    `bson:"name"`
	Time time.Time `bson:"time,omitempty"`
}

// DealDocumentDB is an object of dealDocument(set of pacts) that stores in the DB
// Strings used instead bools to avoid default values confusion
type DealDocumentDB struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Pacts        []PactDB           `bson:"pacts,omitempty"`
	FinalVersion string             `bson:"final_version,omitempty"`
	Judge        SideDB             `bson:"judge,omitempty"`
	Status       []Status           `bson:"status,omitempty"`
	Winner       string             `bson:"winner,omitempty"`
	Blamed       string             `bson:"blamed,omitempty"`
	Type         string             `bson:"type,omitempty"`
	Completed    bool               `bson:"completed,omitempty"` // For now deal will be completed only when judge made his decision
	JusticeCount int                `bson:"justice_count,omitempty"`
}

func (dealDoc DealDocumentDB) getCurrentPact() (PactDB, error) {
	for _, pact := range dealDoc.Pacts {
		if pact.Version == dealDoc.FinalVersion {
			return pact, nil
		}
	}
	return PactDB{}, fmt.Errorf("Deal doesn't have pact for version %s", dealDoc.FinalVersion)
}

func (dealDoc DealDocumentDB) getStatus() (string, error) {
	if len(dealDoc.Status) == 0 {
		return "", errors.New("Status is empty")
	}
	statusOb := dealDoc.Status[len(dealDoc.Status)-1]
	if len(statusOb.Name) == 0 {
		return "", errors.New("Last status is invalid")
	}
	return statusOb.Name, nil
}

// getSuccess counts success over all completed deals since their state can change
func (user UserDB) getSuccess(ctx context.Context, dealTable *mongo.Collection) (int, error) {
	// Go over all deals from deals_result, take each and get result from it
	var success int
	for _, dID := range user.DealResults {
		dealSuccess, err := getUserSuccess(ctx, user.ID.Hex(), dID, dealTable)
		if err != nil {
			return 0, err
		}
		success += dealSuccess
	}
	fmt.Println("[Rating]: To count success for user " + user.ID.Hex() + " used " + strconv.Itoa(len(user.DealResults)) + " deals. Result: " + strconv.Itoa(success))
	return success, nil
}

// getUserSuccess returns 2 or -2 points depending on what state of deal and status of blaming
func getUserSuccess(ctx context.Context, userID, dealID string, dealTable *mongo.Collection) (int, error) {
	deal, err := GetDealDocByIdDB(ctx, dealID, dealTable)
	if err != nil {
		return 0, fmt.Errorf("Failed to get deal %s from DB, err: %v", dealID, err)
	}
	if deal == nil {
		return 0, fmt.Errorf("Invalid operation, deal %s doesn't exist or not completed yet, err: %v", dealID, err)
	}
	// No need to count it yet
	if !deal.Completed {
		return 0, nil
	}
	pact, err := deal.getCurrentPact()
	if err != nil {
		return 0, fmt.Errorf("Failed to get pact from deal %s, err: %v", dealID, err)
	}
	isRed, isUnknown := 1, true
	for _, rP := range pact.Red.Participants {
		if rP.ID == userID {
			isRed = 1
			isUnknown = false
			break
		}
	}
	// We can't be sure user has this deal, we need to check it for sure
	if isUnknown {
		for _, bP := range pact.Blue.Participants {
			if bP.ID == userID {
				isRed = -1
				isUnknown = false
				break
			}
		}
	}
	// It user still unknown
	if isUnknown {
		return 0, fmt.Errorf("User %s side can't be found in deal %s", userID, dealID)
	}
	var redWon int
	if deal.Winner == "red" {
		redWon = 1
	}
	if deal.Winner == "blue" {
		redWon = -1
	}
	if !(redWon == 1 || redWon == -1) {
		return 0, fmt.Errorf("Failed to get deal %s winner correctly, deal winner: %s", dealID, deal.Winner)
	}

	var notBlamed int
	if deal.Blamed == "No" {
		notBlamed = 1
	}
	if deal.Blamed == "Yes" {
		notBlamed = -1
	}
	if !(notBlamed == 1 || notBlamed == -1) {
		return 0, fmt.Errorf("Failed to get deal blame status correctly, deal blame status: %s", deal.Blamed)
	}

	// formula: {sideRed(1 or -1)} * {redWon(1 or -1)} * 2 points * -{blamed(1 or -1)}
	return redWon * isRed * notBlamed * 2, nil
}

// getSuccess counts justice over all completed deals since their state can change
func (user UserDB) getJustice(ctx context.Context, dealTable *mongo.Collection) (int, error) {
	// Go over all deals from deals_result, take each and get result from it
	justice := 0
	if user.JudgeProfile == nil || !user.IsJudge {
		return 0, errors.New("Can't count justice of common user")
	}
	deals, err := GetCompletedDeals(ctx, dealTable)
	if err != nil {
		return 0, fmt.Errorf("Can't get completed deals from DB")
	}
	for _, decision := range user.JudgeProfile.Decisions {
		for _, completedDeal := range deals {
			if decision.DealID == completedDeal.ID.Hex() {
				dealJustice, err := getJudgeJustice(ctx, user.ID.Hex(), *completedDeal)
				if err != nil {
					return 0, err
				}
				justice += dealJustice
			}
		}
	}
	return justice, nil
}

// getJudgeJustice takes current status of deal and compares it with judge decision, return 2 or -4 points
func getJudgeJustice(ctx context.Context, judgeID string, deal DealDocumentDB) (int, error) {
	participating := false
	for _, j := range deal.Judge.Participants {
		if judgeID == j.ID {
			participating = true
			break
		}
	}
	if !participating {
		return 0, fmt.Errorf("Confusion, judge %s don't participate in deal %s", judgeID, deal.ID.Hex())
	}
	// TODO: we don't care about judge own decision because for now only one judge is possible. So judge decision is
	// the winner set in the deal. Once we have many judges we have to compare judge decision and deal winner
	justiceBonus := 0
	if deal.Blamed == "Yes" {
		justiceBonus = -4
	}
	if deal.Blamed == "No" {
		justiceBonus = 2
	}
	if justiceBonus == 0 {
		return 0, fmt.Errorf("Invalid data, can't get blame status in deal %s", deal.ID.Hex())
	}
	return justiceBonus, nil
}

// ConvertUserToDB converts user from pb.User type to UserDB type
func ConvertUserToDB(user *pb.User) (*UserDB, error) {
	if user == nil {
		return nil, fmt.Errorf("User is not valid")
	}
	userResp := &UserDB{
		Name:          user.GetName(),
		Surname:       user.GetSurname(),
		Username:      user.GetUsername(),
		DealDocs:      user.GetDealDocs(),
		Accepted:      user.GetAccepted(),
		Offerings:     user.GetOfferings(),
		Participating: user.GetParticipating(),
		IsJudge:       user.GetIsJudge(),
	}
	if len(user.GetId()) > 0 {
		userID, err := primitive.ObjectIDFromHex(user.GetId())
		if err != nil {
			return nil, fmt.Errorf("Invalid user, bad id %q", user.GetId())
		}
		userResp.ID = userID
	}
	if user.GetJudgeProfile() != nil {
		judgeProfile := user.GetJudgeProfile()
		userResp.JudgeProfile = &JudgeProfile{
			Propositions:   judgeProfile.GetPropositions(),
			Participatings: judgeProfile.GetParticipatings(),
		}
		for _, d := range judgeProfile.GetDecisions() {
			userResp.JudgeProfile.Decisions = append(userResp.JudgeProfile.Decisions, Decision{
				DealID: d.GetDealId(),
				Winner: d.GetWinner(),
				When:   d.GetWhen(),
			})
		}
	}

	return userResp, nil
}

// ConvertDBToUser converts user from *UserDB type to *pb.User type
func ConvertDBToUser(user *UserDB) (*pb.User, error) {
	if user == nil {
		return nil, fmt.Errorf("User is not valid")
	}
	userResp := &pb.User{
		Name:          user.Name,
		Surname:       user.Surname,
		Username:      user.Username,
		DealDocs:      user.DealDocs,
		Accepted:      user.Accepted,
		Offerings:     user.Offerings,
		Participating: user.Participating,
		Id:            user.ID.Hex(),
		IsJudge:       user.IsJudge,
	}
	if user.JudgeProfile != nil {
		userResp.JudgeProfile = &pb.JudgeProfile{
			Propositions:   user.JudgeProfile.Propositions,
			Participatings: user.JudgeProfile.Participatings,
		}
		for _, d := range user.JudgeProfile.Decisions {
			userResp.JudgeProfile.Decisions = append(userResp.JudgeProfile.Decisions, &pb.Decision{
				DealId: d.DealID,
				Winner: d.Winner,
				When:   d.When,
			})
		}
	}
	for _, dID := range user.DealResults {
		userResp.DealResults = append(userResp.DealResults, dID)
	}
	return userResp, nil
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
	es = append(es, bson.E{Key: "deal_docs", Value: u.DealDocs})
	es = append(es, bson.E{Key: "offerings", Value: u.Offerings})
	es = append(es, bson.E{Key: "accepted", Value: u.Accepted})
	es = append(es, bson.E{Key: "participating", Value: u.Participating})
	es = append(es, bson.E{Key: "deal_results", Value: u.DealResults})
	if u.JudgeProfile != nil {
		es = append(es, bson.E{Key: "judge_profile.participatings", Value: u.JudgeProfile.Participatings})
		es = append(es, bson.E{Key: "judge_profile.propositions", Value: u.JudgeProfile.Propositions})
		es = append(es, bson.E{Key: "judge_profile.decisions", Value: u.JudgeProfile.Decisions})
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
	if len(dd.Status) > 0 {
		es = append(es, bson.E{Key: "status", Value: dd.Status})
	}
	if len(dd.Winner) > 0 {
		es = append(es, bson.E{Key: "winner", Value: dd.Winner})
	}
	if len(dd.Blamed) > 0 {
		es = append(es, bson.E{Key: "blamed", Value: dd.Blamed})
	}
	if len(dd.Type) > 0 {
		es = append(es, bson.E{Key: "type", Value: dd.Type})
	}
	es = append(es, bson.E{Key: "completed", Value: dd.Completed})
	es = append(es, bson.E{Key: "justice_count", Value: dd.JusticeCount})
	return es
}

func CreateUserDB(ctx context.Context, user UserDB, table *mongo.Collection) (string, error) {
	res, err := table.InsertOne(ctx, user)
	if err != nil {
		fmt.Println("Error creating user in mongo: ", err)
	}
	return res.InsertedID.(primitive.ObjectID).Hex(), err
}

// GetUserByIDDB returns user from DB by it's id
func GetUserByIDDB(ctx context.Context, userId string, table *mongo.Collection) (*UserDB, error) {
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

// GetJudges returns all judges (users that has `isJudge` property true)
func GetJudges(ctx context.Context, table *mongo.Collection) ([]*UserDB, error) {
	judges := []*UserDB{}

	cursor, err := table.Find(ctx, bson.D{{Key: "is_judge", Value: true}})
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		j := &UserDB{}
		if err := cursor.Decode(j); err != nil {
			fmt.Println("Error getting judges from mongo: ", err)
			return nil, err
		}
		judges = append(judges, j)
	}

	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, nil
		}
		fmt.Println("Error getting judges from mongo: ", err)
		return nil, err
	}

	return judges, nil
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

func GetCompletedDeals(ctx context.Context, table *mongo.Collection) ([]*DealDocumentDB, error) {
	// Get all deals is no watching deals
	cursor, err := table.Find(ctx, bson.D{{Key: "completed", Value: true}})
	if err != nil {
		fmt.Println("Error getting deals from DB: ", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	deals := make([]*DealDocumentDB, 0)
	for cursor.Next(ctx) {
		d := &DealDocumentDB{}
		if err := cursor.Decode(d); err != nil {
			fmt.Println("Error getting deals from DB: ", err)
			return nil, err
		}
		deals = append(deals, d)
	}
	return deals, err
}

func GetDealDocByIdDBConvert(ctx context.Context, dealDocID string, table *mongo.Collection) (*pb.DealDocument, error) {
	dealDocDB, err := GetDealDocByIdDB(ctx, dealDocID, table)
	if err != nil {
		return nil, err
	}

	dealDocumentRes := &pb.DealDocument{
		Id:           dealDocDB.ID.Hex(),
		FinalVersion: dealDocDB.FinalVersion,
		Winner:       dealDocDB.Winner,
		Blamed:       dealDocDB.Blamed,
		JusticeCount: int64(dealDocDB.JusticeCount),
		Type:         dealDocDB.Type,
	}
	status, err := dealDocDB.getStatus()
	if err != nil {
		return nil, err
	}
	dealDocumentRes.Status = status
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

// DeleteUserByIDDB deletes user `userId` from DB table `table`
func DeleteUserByIDDB(ctx context.Context, userId string, table *mongo.Collection) (*UserDB, error) {
	userIDDB, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		fmt.Println("Error creating object id to get user: ", err)
		return nil, err
	}
	userDB := &UserDB{}

	err = table.FindOneAndDelete(ctx, bson.D{{Key: "_id", Value: userIDDB}}).Decode(userDB)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, nil
		}
		fmt.Println("Error getting user from mongo: ", err)
		return nil, err
	}

	return userDB, err
}

// GetUserByUsernameDB returns UserDB by username if it exists
func GetUserByUsernameDB(ctx context.Context, username string, table *mongo.Collection) (*UserDB, string, error) {
	userDB := &UserDB{}
	err := table.FindOne(ctx, bson.D{{Key: "username", Value: username}}).Decode(userDB)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, "", nil
		}
		fmt.Println("Error getting user from mongo: ", err)
		return nil, "", err
	}

	return userDB, userDB.ID.Hex(), err
}

// UpdateUserDB updates user in DB using {userID} to find it and user to update data
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

	user, err := GetUserByIDDB(ctx, userID, userTable)
	if err != nil || len(user.Username) == 0 {
		err = fmt.Errorf("Failed to get user by %q id", userID)
		fmt.Println("[ERROR]: ", err.Error())
		return err
	}
	userAccepted, err := userAcceptDeal(user, dealDocID)
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
	dealDoc, err = offerDealForSide(dealDoc, side, userID)
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
		pactSide.Participants = append(pactSide.Participants, ParticipantDB{
			ID:       userID,
			Accepted: false,
		})
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

// Update {userID} user acceptance status in deal document
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

// Move deal in user [Offerings] -> [Accepted]
func userAcceptDeal(user *UserDB, dealID string) (*UserDB, error) {
	var err error
	resUser := user
	if user == nil {
		err = fmt.Errorf("Invalid user in input")
		return nil, err
	}
	initOfferLen := len(user.Offerings)
	for i, offerID := range user.Offerings {
		if offerID == dealID {
			resUser.Offerings = append(user.Offerings[:i], user.Offerings[i+1:]...)
			break
		}
	}
	if initOfferLen == len(user.Offerings) {
		err = fmt.Errorf("Failed to find offer %q in user offers", dealID)
		return nil, err
	}
	resUser.Accepted = append(user.Accepted, dealID)
	return resUser, err
}

// CheckToWatchDeal checks whether it's needed to send deal `dealID` to the watcher to watch it's timeout
func CheckToWatchDeal(ctx context.Context, dealDocID string, dealDocTable *mongo.Collection) (bool, error) {
	// Get deal document
	dealDoc, err := GetDealDocByIdDB(ctx, dealDocID, dealDocTable)
	if err != nil {
		fmt.Println("Failed to get deal document: ", err)
		return false, err
	}
	// Check whether deal document accepted by everyone
	isDealDocAcceptedByUsers, err := isDealDocumentAcceptedByUsers(dealDoc)
	if err != nil {
		fmt.Println("Failed to check acceptance of deal document: ", err)
		return false, err
	}
	return isDealDocAcceptedByUsers, err
}

// JudgeAcceptDeal sets judge (only one possible) to the deal judge and update deal status
func JudgeAcceptDeal(ctx context.Context, judgeID, dealID string, dealDocTable *mongo.Collection) error {
	// Get deal document
	dealDocIDDB, err := primitive.ObjectIDFromHex(dealID)
	if err != nil {
		fmt.Println("Error creating object id to get deal document: ", err)
		return err
	}
	deal, err := GetDealDocByIdDB(ctx, dealID, dealDocTable)
	if err != nil {
		fmt.Println("Failed to get deal "+dealID+": ", err)
		return err
	}
	deal.Judge = SideDB{
		Type: pb.SideType_JUDGE,
		Participants: []ParticipantDB{
			ParticipantDB{
				ID:       judgeID,
				Accepted: true,
			},
		},
	}
	_, err = dealDocTable.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: dealDocIDDB}},
		bson.D{{"$set", deal.toMongoFormat()}},
	)
	if err != nil {
		return fmt.Errorf("Failed to update deal %s judge %s, err: %v", dealID, judgeID, err)
	}
	err = UpdateDealStatus(ctx, dealID, "ALL_ACCEPTED", dealDocTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to update deal "+dealID+" status: ", err)
		return err
	}
	return err
}

// UpdateDealStatus updates deal document `dealDocID` status to `status`
func UpdateDealStatus(ctx context.Context, dealDocID string, status string, dealDocTable *mongo.Collection) error {
	// Get deal document
	dealDocIDDB, err := primitive.ObjectIDFromHex(dealDocID)
	if err != nil {
		fmt.Println("Error creating object id to get deal document: ", err)
		return err
	}
	deal, err := GetDealDocByIdDB(ctx, dealDocID, dealDocTable)
	if err != nil {
		fmt.Println("Failed to get deal "+dealDocID+": ", err)
		return err
	}
	deal.Status = append(deal.Status, Status{
		Name: status,
		Time: time.Now(),
	})
	_, err = dealDocTable.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: dealDocIDDB}},
		bson.D{{"$set", []bson.E{bson.E{Key: "status", Value: deal.Status}}}},
	)
	return err
}

// UpdateDeal updates deal document `dealDocID`
func UpdateDeal(ctx context.Context, dealDoc DealDocumentDB, dealDocTable *mongo.Collection) error {
	// Get deal document
	_, err := GetDealDocByIdDB(ctx, dealDoc.ID.Hex(), dealDocTable)
	if err != nil {
		fmt.Println("Failed to get deal "+dealDoc.ID.Hex()+", error: ", err)
		return err
	}
	_, err = dealDocTable.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: dealDoc.ID}},
		bson.D{{"$set", dealDoc.toMongoFormat()}},
	)
	return err
}

// MakeDecision appends to judge decisions deal {dealDocID} with winner and removes deal {dealDocID} from {participatings}
func MakeDecision(ctx context.Context, judge *UserDB, dealDocID, winner string, userTable *mongo.Collection) error {
	// Get deal document
	dealIndex := -1

	for i, d := range judge.JudgeProfile.Participatings {
		if d == dealDocID {
			dealIndex = i
			break
		}
	}
	if dealIndex == -1 {
		fmt.Println("{DEBUG}", "Judge "+judge.ID.Hex()+" doesn't participate in "+dealDocID+" deal")
		return errors.New("Judge " + judge.ID.Hex() + " doesn't participate in " + dealDocID + " deal")
	}
	judge.JudgeProfile.Decisions = append(judge.JudgeProfile.Decisions, Decision{
		DealID: dealDocID,
		Winner: winner,
		When:   time.Now().String(),
	})
	judge.JudgeProfile.Participatings = append(judge.JudgeProfile.Participatings[:dealIndex], judge.JudgeProfile.Participatings[dealIndex+1:]...)
	_, err := userTable.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: judge.ID}},
		bson.D{{"$set", judge.toMongoFormat()}},
	)
	return err
}

// SetDealWinner sets deal {dealDocID} winner and updates it's status to WINNER_SET
func SetDealWinner(ctx context.Context, dealDocID, winner string, dealDocTable, userTable *mongo.Collection) error {
	// Get deal document
	dealDocIDDB, err := primitive.ObjectIDFromHex(dealDocID)
	if err != nil {
		fmt.Println("Error creating object id to get deal document: ", err)
		return err
	}
	deal, err := GetDealDocByIdDB(ctx, dealDocID, dealDocTable)
	if err != nil {
		fmt.Println("Failed to get deal "+dealDocID+": ", err)
		return err
	}
	if deal.Completed {
		return fmt.Errorf("Deal %s already completed, can't change decision", dealDocID)
	}
	deal.Status = append(deal.Status, Status{
		Name: "WINNER_SET",
		Time: time.Now(),
	})
	deal.Blamed = "No"
	deal.Winner = winner
	deal.Completed = true
	// Get judge(judges in future), count their justice and set justiceCount for this deal to know how much justice need to blame it
	if len(deal.Judge.Participants) == 0 {
		return fmt.Errorf("Invalid data in the deal %s, no judges", dealDocID)
	}
	judgeID := deal.Judge.Participants[0].ID
	if len(judgeID) == 0 {
		return fmt.Errorf("Invalid data in the deal %s, judge id can't be empty", dealDocID)
	}
	judge, err := GetUserByIDDB(ctx, judgeID, userTable)
	if err != nil {
		return fmt.Errorf("Failed to get judge %s from DB, err: %v", judgeID, err)
	}
	justiceCount, err := judge.getJustice(ctx, dealDocTable)
	if err != nil {
		return fmt.Errorf("Failed to get judge %s justice count, err: %v", judgeID, err)
	}
	deal.JusticeCount = justiceCount
	_, err = dealDocTable.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: dealDocIDDB}},
		bson.D{{"$set", deal.toMongoFormat()}},
	)
	return err
}

func isDealDocumentAcceptedByUsers(dealDoc *DealDocumentDB) (isDealDocAccepted bool, err error) {
	var pact *PactDB
	for i, pactTmp := range dealDoc.Pacts {
		if pactTmp.Version == dealDoc.FinalVersion {
			pact = &dealDoc.Pacts[i]
		}
	}
	if pact == nil {
		err = errors.New("Deal document is invalid, can't find needed pact")
		fmt.Println("[ERROR]: ", err.Error())
		return false, err
	}
	// Check blue side
	if len(pact.Blue.Participants) == 0 {
		err = fmt.Errorf("Deal document has no participants on Blue side")
		fmt.Println("[ERROR]: ", err.Error())
		return false, err
	}
	for _, participant := range pact.Blue.Participants {
		if participant.Accepted == false {
			return false, nil
		}
	}

	// Check red side
	if len(pact.Red.Participants) == 0 {
		err = fmt.Errorf("Deal document has no participants on Red side")
		fmt.Println("[ERROR]: ", err.Error())
		return false, err
	}
	for _, participant := range pact.Red.Participants {
		if participant.Accepted == false {
			return false, nil
		}
	}
	return true, nil
}

// TellUserDealStarted change user stats that shows that they are participating in deal
func TellUserDealStarted(ctx context.Context, dealDoc DealDocumentDB, userTable *mongo.Collection) error {
	// Get current pact
	pact, err := dealDoc.getCurrentPact()
	if err != nil {
		return fmt.Errorf("Failed to get pact: %s", err.Error())
	}
	// Get all participants
	allParticipants := append(pact.Blue.Participants, pact.Red.Participants...)
	for _, p := range allParticipants {
		// Find user
		user, err := GetUserByIDDB(ctx, p.ID, userTable)
		if err != nil {
			return fmt.Errorf("Failed to get user from pact data: %s", err.Error())
		}
		// Update user stats
		resUser := *user
		for i, dealID := range user.Accepted {
			if dealDoc.ID.Hex() == dealID {
				resUser.Accepted = append(user.Accepted[:i], user.Accepted[i+1:]...)
				resUser.Participating = append(user.Participating, dealID)
				break
			}
		}
		if len(resUser.Accepted) == len(user.Accepted) {
			return fmt.Errorf("User %s don't accept deal %s", user.ID, dealDoc.ID.Hex())
		}
		// Save user
		fmt.Printf("User %q participating in %v has accepted %v", resUser.Username, resUser.Participating, resUser.Accepted)
		err = UpdateUserDB(ctx, p.ID, &resUser, userTable)
		if err != nil {
			return fmt.Errorf("Failed to update user %s: %s", p.ID, err.Error())
		}
	}
	return nil
}
