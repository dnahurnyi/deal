//
// Copyright 2019
//
// @author: Denys Nahurnyi
// @email:  dnahurnyi@gmail.com
// ---------------------------------------------------------------------------
package dataSvc

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"strconv"
	"time"

	grpcutils "github.com/DenysNahurnyi/deal/common/grpc"
	"github.com/DenysNahurnyi/deal/pb/generated/pb"
	"github.com/go-kit/kit/log"
	"github.com/mongodb/mongo-go-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service interface {
	CreateUser(ctx context.Context, user *UserDB) (string, error)
	GetUser(ctx context.Context, userID string) (*UserDB, error)
	DeleteUser(ctx context.Context, userID string) (*UserDB, error)
	UpdateUser(ctx context.Context, user *UserDB) (*UserDB, error)
	CreateDealDocument(ctx context.Context, userID string, dealDocument *pb.Pact) (string, error)
	CreateBlameDocument(ctx context.Context, userID, blamedDealID, blameReason string) (string, error)
	GetDealDocument(ctx context.Context, dealDocumentID string) (*pb.DealDocument, error)
	DealTimeout(ctx context.Context, dealDocID string) error
	OfferDealDocument(ctx context.Context, dealDocId, username string, toJudge bool) error
	AcceptDealDocument(ctx context.Context, userID, dealDocId string, side pb.SideType) error
	OfferJudges(ctx context.Context, dealDocId string) error
	JudgeAccept(ctx context.Context, judgeID, dealDocId string) error
	JudgeDecide(ctx context.Context, judgeID, dealDocID, redWon string) error
	ActivateBlame(ctx context.Context, judgeID, blameID string) error
	JoinBlame(ctx context.Context, userID, blameID string) error
	GetPubKey() *rsa.PublicKey
	getDealsTable() *mongo.Collection
}

type service struct {
	envType          string
	userTable        *mongo.Collection
	dealDocTable     *mongo.Collection
	authSvcClient    pb.AuthServiceClient
	watcherSvcClient pb.WatcherServiceClient
	uKey             *rsa.PublicKey
}

func NewService(logger log.Logger, mgc *mongo.Client, authSvcClient *pb.AuthServiceClient, watcherSvcClient *pb.WatcherServiceClient) (Service, error) {
	userTable := mgc.Database("travel").Collection("users")
	dealDocTable := mgc.Database("travel").Collection("dealDocuments")
	ctx := context.Background()
	authSvcClientValue := *authSvcClient
	getPubKeyResp, err := authSvcClientValue.GetCheckTokenKey(ctx, &pb.EmptyReq{
		ReqHdr: &pb.ReqHdr{
			Tid: "call to get pub key",
		},
	})
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get pub key from authSvc, err:", err)
		return nil, err
	}
	uKey, err := grpcutils.CreatePubKey(getPubKeyResp.GetNBase64(), int(getPubKeyResp.GetE()))
	if err != nil {
		fmt.Println("[LOG]:", "Failed to create pub key for authSvc tokens, err:", err)
		return nil, err
	}
	fmt.Println("Data svc uKey: ", uKey)
	watcherSvcClientValue := *watcherSvcClient

	return &service{
		envType:          "test",
		userTable:        userTable,
		dealDocTable:     dealDocTable,
		authSvcClient:    authSvcClientValue,
		watcherSvcClient: watcherSvcClientValue,
		uKey:             uKey,
	}, nil
}

func (s *service) CreateUser(ctx context.Context, userReq *UserDB) (string, error) {
	userGet, _, err := GetUserByUsernameDB(ctx, userReq.Username, s.userTable)
	if err != nil {
		fmt.Println("Failed to get user from DB")
		return "", err
	}
	if userGet != nil && len(userGet.Username) > 0 {
		fmt.Println("[WARNING] user already exist")
		return "", errors.New("User already exist")
	}
	// Can't just use userReq because attacker can create it with participatin deals
	userID, err := CreateUserDB(ctx, UserDB{
		Name:     userReq.Name,
		Surname:  userReq.Surname,
		Username: userReq.Username,
	}, s.userTable)
	return userID, err
}

func (s *service) GetUser(ctx context.Context, userID string) (*UserDB, error) {
	return GetUserByIDDB(ctx, userID, s.userTable)
}

func (s *service) DeleteUser(ctx context.Context, userID string) (*UserDB, error) {
	user, err := DeleteUserByIDDB(ctx, userID, s.userTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to delete user, err: ", err)
		return nil, err
	}
	_, err = s.authSvcClient.DeleteUser(ctx, &pb.DeleteSecureUserReq{
		ReqHdr: &pb.ReqHdr{
			Tid: "some transaction id to delete user in auth",
		},
		TokenId: userID,
	})
	if err != nil {
		fmt.Println("[LOG]:", "Failed to delete user in auth service, err: ", err)
		return nil, err
	}
	return user, nil
}

func (s *service) GetPubKey() *rsa.PublicKey {
	return s.uKey
}

func (s *service) getDealsTable() *mongo.Collection {
	return s.dealDocTable
}

func (s *service) UpdateUser(ctx context.Context, user *UserDB) (*UserDB, error) {
	userExist, err := GetUserByIDDB(ctx, user.ID.Hex(), s.userTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get user, err: ", err)
		return nil, err
	}
	if len(userExist.Username) == 0 {
		fmt.Println("[WARNING] user doesn't exist")
		return nil, errors.New("User doesn't exist")
	}
	// Update user common props
	{
		userExist.Name = user.Name
		userExist.Surname = user.Surname
		userExist.Username = user.Username
	}
	err = UpdateUserDB(ctx, user.ID.Hex(), userExist, s.userTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to update user in data service, err: ", err)
		return nil, err
	}
	return user, nil
}

func (s *service) CreateBlameDocument(ctx context.Context, userID, blamedDealID, blameReason string) (string, error) {
	userDB, err := GetUserByIDDB(ctx, userID, s.userTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get user from DB, err: ", err)
		return "", err
	}
	if !userDB.IsJudge {
		return "", fmt.Errorf("Invalid operation, user %s can't blame because he is not judge", userID)
	}
	userJustice, err := userDB.getJustice(ctx, s.dealDocTable)
	if err != nil {
		return "", fmt.Errorf("Failed to get user %s justice, err: %v", userID, err)
	}
	if userJustice < 0 {
		return "", fmt.Errorf("User %s can't join deal because his justice level is toxic", userID)
	}
	if userJustice == 0 {
		return "", fmt.Errorf("User %s can't join deal because his justice level is useless", userID)
	}
	blameDocumentDB, err := createInitBlameDocument(userID, blamedDealID, blameReason, "BLAME", userJustice)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to create blame document, err: ", err)
		return "", err
	}
	blameDocID, err := CreateDealDocumentDB(ctx, blameDocumentDB, s.dealDocTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to create deal document, err: ", err)
		return "", err
	}
	userDB.DealDocs = append(userDB.DealDocs, blameDocID)
	// This is irreversible so once you created it, you participate in it
	userDB.Participating = append(userDB.Participating, blameDocID)

	err = UpdateUserDB(ctx, userID, userDB, s.userTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to add deal document to users, err: ", err)
		return "", err
	}
	return blameDocID, err
}

func (s *service) CreateDealDocument(ctx context.Context, userID string, dealDocument *pb.Pact) (string, error) {
	dealDocumentDB, err := createInitDealDocument(userID, dealDocument.GetContent(), dealDocument.GetTimeout(), "COMMON")
	if err != nil {
		fmt.Println("[LOG]:", "Failed to create deal document, err: ", err)
		return "", err
	}
	dealDocID, err := CreateDealDocumentDB(ctx, dealDocumentDB, s.dealDocTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to create deal document, err: ", err)
		return "", err
	}
	userDB, err := GetUserByIDDB(ctx, userID, s.userTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get user from DB, err: ", err)
		return "", err
	}
	userDB.DealDocs = append(userDB.DealDocs, dealDocID)
	userDB.Accepted = append(userDB.Accepted, dealDocID)

	err = UpdateUserDB(ctx, userID, userDB, s.userTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to add deal document to users, err: ", err)
		return "", err
	}
	return dealDocID, err
}

func createInitDealDocument(redUserID, content, timeout, docType string) (DealDocumentDB, error) {
	// Checks
	if len(redUserID) == 0 {
		fmt.Println("[LOG] Invalid input, userID is invalid")
		return DealDocumentDB{}, errors.New("Invalid input, userID is invalid: " + redUserID)
	}
	if len(content) == 0 {
		fmt.Println("[LOG] Invalid input, content is invalid")
		return DealDocumentDB{}, errors.New("Invalid input, content is invalid: " + content)
	}
	if len(timeout) == 0 {
		fmt.Println("[LOG] Invalid input, timeout is invalid")
		return DealDocumentDB{}, errors.New("Invalid input, timeout is invalid: " + timeout)
	}
	if len(docType) == 0 {
		fmt.Println("[LOG] Invalid input, docType is invalid")
		return DealDocumentDB{}, errors.New("Invalid input, docType is invalid: " + docType)
	}

	redSide := SideDB{
		Type: pb.SideType_RED,
		Participants: []ParticipantDB{
			ParticipantDB{
				ID:       redUserID,
				Accepted: true,
			},
		},
	}
	firstPact := PactDB{
		Content: content,
		Red:     redSide,
		Blue: SideDB{
			Type:         pb.SideType_BLUE,
			Participants: []ParticipantDB{},
		},
		Version: "initial(#1)",
		Timeout: timeout,
	}
	return DealDocumentDB{
		Type:         docType,
		Pacts:        []PactDB{firstPact},
		FinalVersion: firstPact.Version,
		Status: []Status{
			Status{
				Name: "INITIAL DEAL STAGE",
				Time: time.Now(),
			},
		},
	}, nil
}

func createInitBlameDocument(redUserID, blamedDealID, content, docType string, justiceCount int) (DealDocumentDB, error) {
	// Checks
	if len(redUserID) == 0 {
		fmt.Println("[LOG] Invalid input, userID is invalid")
		return DealDocumentDB{}, errors.New("Invalid input, userID is invalid: " + redUserID)
	}
	// In case of blame deal blue side will contain deal ID as participant and it will accept deal autonatically
	if len(blamedDealID) == 0 {
		fmt.Println("[LOG] Invalid input, blamedDealID is invalid")
		return DealDocumentDB{}, errors.New("Invalid input, blamedDealID is invalid: " + blamedDealID)
	}
	if justiceCount <= 0 {
		fmt.Println("[LOG] Invalid input, justiceCount is invalid")
		return DealDocumentDB{}, errors.New("Invalid input, justiceCount is invalid: " + strconv.Itoa(justiceCount))
	}
	if len(content) == 0 {
		fmt.Println("[LOG] Invalid input, content is invalid")
		return DealDocumentDB{}, errors.New("Invalid input, content is invalid: " + content)
	}
	if len(docType) == 0 {
		fmt.Println("[LOG] Invalid input, docType is invalid")
		return DealDocumentDB{}, errors.New("Invalid input, docType is invalid: " + docType)
	}

	redSide := SideDB{
		Type: pb.SideType_RED,
		Participants: []ParticipantDB{
			ParticipantDB{
				ID:       redUserID,
				Accepted: true,
			},
		},
	}
	blueSide := SideDB{
		Type: pb.SideType_BLUE,
		Participants: []ParticipantDB{
			ParticipantDB{
				ID:       blamedDealID,
				Accepted: true,
			},
		},
	}
	firstPact := PactDB{
		Content: content,
		Red:     redSide,
		Blue:    blueSide,
		Version: "initial(#1)",
	}
	return DealDocumentDB{
		Type:         docType,
		Pacts:        []PactDB{firstPact},
		FinalVersion: firstPact.Version,
		JusticeCount: justiceCount,
		Judge: SideDB{
			Type: pb.SideType_JUDGE,
			Participants: []ParticipantDB{
				ParticipantDB{
					ID:       redUserID,
					Accepted: true,
				},
			},
		},
		Status: []Status{
			Status{
				Name: "INITIAL BLAME DEAL STAGE",
				Time: time.Now(),
			},
		},
	}, nil
}

// OfferDealDocument offer another user deal document. If toJudge true, then it is offer to user with `username` to judge this deal, in another case it's offer to participate in the deal
func (s *service) OfferDealDocument(ctx context.Context, dealDocID, username string, toJudge bool) error {
	dealDoc, err := GetDealDocByIdDB(ctx, dealDocID, s.dealDocTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get deal document from DB, err: ", err)
		return err
	}
	if dealDoc == nil {
		fmt.Println("[LOG]:", "Deal document doesn't exist")
		return errors.New("Deal document doesn't exist")
	}

	offeredUser, offeredUserID, err := GetUserByUsernameDB(ctx, username, s.userTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get user from DB, err: ", err)
		return err
	}
	if len(offeredUserID) == 0 {
		fmt.Println("[WARNING] user doesn't exist")
		return errors.New("User doesn't exist")
	}
	var offerPersonSide pb.SideType
	// To reduce repeated code it has to be one interface that gather User and Judge
	if toJudge {
		// This functionality has to be moved to another method

		// // Check in the judge records
		// if !offeredUser.IsJudge {
		// 	return status.Errorf(codes.InvalidArgument, "Specified user is not judge")
		// }
		// if offeredUser.JudgeProfile == nil {
		// 	return status.Errorf(codes.Internal, "Specified user has invalid judge record")
		// }

		// for _, d := range offeredUser.JudgeProfile.Offerings {
		// 	if d == dealDocID {
		// 		// This user already has this deal in offerings, but the one who offered shouldn't know that
		// 		return nil
		// 	}
		// }
		// for _, d := range offeredUser.JudgeProfile.Accepted {
		// 	if d == dealDocID {
		// 		// This user already has accepted this deal, but the one who offered shouldn't know that
		// 		return nil
		// 	}
		// }
		// offeredUser.JudgeProfile.Offerings = append(offeredUser.JudgeProfile.Offerings, dealDocID)
		// offerPersonSide = pb.SideType_JUDGE
	} else {
		// Check in the user records
		for _, d := range offeredUser.Offerings {
			if d == dealDocID {
				// This user already has this deal in offerings, but the one who offered shouldn't know that
				return nil
			}
		}
		for _, d := range offeredUser.Accepted {
			if d == dealDocID {
				// This user already has accepted this deal, but the one who offered shouldn't know that
				return nil
			}
		}
		offeredUser.Offerings = append(offeredUser.Offerings, dealDocID)
		offerPersonSide = pb.SideType_BLUE
	}

	err = UpdateUserDB(ctx, offeredUserID, offeredUser, s.userTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to user in DB, err: ", err)
		return err
	}
	return OfferDealDocDB(ctx, dealDocID, offeredUserID, offerPersonSide, s.dealDocTable)
}

func (s *service) GetDealDocument(ctx context.Context, dealDocumentID string) (*pb.DealDocument, error) {
	dealDoc, err := GetDealDocByIdDBConvert(ctx, dealDocumentID, s.dealDocTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get deal document, err: ", err)
		return nil, err
	}
	return dealDoc, err
}

// AcceptDealDocument accepts `dealID` deal for `userID` user
func (s *service) AcceptDealDocument(ctx context.Context, userID, dealDocID string, side pb.SideType) error {
	// Mark document as accepted
	// Get doc to make sure it exists
	dealDoc, err := GetDealDocByIdDB(ctx, dealDocID, s.dealDocTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get deal document from DB, err: ", err)
		return err
	}
	if dealDoc == nil {
		fmt.Println("[LOG]:", "Deal document doesn't exist")
		return errors.New("Deal document doesn't exist")
	}
	err = AcceptDealDocDB(ctx, dealDocID, userID, side, s.dealDocTable, s.userTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to accept deal: ", err)
		return err
	}
	// Whethere it's accept deal action, maybe everyone accepted deal so we could run watchDeal on watcherSvc
	isDealDocAcceptedByUsers, err := CheckToWatchDeal(ctx, dealDocID, s.dealDocTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to check wheter deal accepted by every participant: ", err)
		return err
	}
	if isDealDocAcceptedByUsers {
		// Update deal status
		err = UpdateDealStatus(ctx, dealDocID, "ACCEPTED_BY_USERS", s.dealDocTable)
		if err != nil {
			fmt.Println("[LOG]:", "Failed to deal status: ", err)
			return err
		}
		// Find the judge
		err = s.OfferJudges(ctx, dealDocID)
		if err != nil {
			fmt.Println("[LOG]:", "Failed to offer the deal : "+dealDocID+" for judges: ", err)
			return err
		}
		// Move this functionality to the independent method s.SendDealToWatcher

		// Send deal to watcher
		// _, err := s.watcherSvcClient.HoldAndWatch(ctx, &pb.HoldAndWatchReq{
		// 	ReqHdr: &pb.ReqHdr{
		// 		Tid: "1234",
		// 	},
		// 	DealId: dealDocID,
		// })
		// if err != nil {
		// 	fmt.Println("[LOG]:", "Failed to watch new deal: ", err)
		// 	return err
		// }
		// // Update user deal status
		// err = TellUserDealStarted(ctx, *dealDoc, s.userTable)
		// if err != nil {
		// 	fmt.Printf("[LOG]: Failed to update user statuses to [PARTICIPATING] in deal %s: %s\n", dealDoc.ID, err)
		// 	return err
		// }
	}
	return err
}

func (s *service) OfferJudges(ctx context.Context, dealDocID string) error {
	// Get all judges
	judges, err := GetJudges(ctx, s.userTable)
	fmt.Println("judges, err: ", judges, err)
	// Update propositions
	for _, j := range judges {
		if j.JudgeProfile == nil {
			fmt.Println("[LOG]:", "Invalid judge profile data: ", err)
			return status.Errorf(codes.InvalidArgument, "Judge with id "+j.ID.Hex()+" has invalid judge profile")
		}
		alreadyOffered := false
		for _, p := range j.JudgeProfile.Propositions {
			if p == dealDocID {
				alreadyOffered = true
				break
			}
		}
		if !alreadyOffered {
			j.JudgeProfile.Propositions = append(j.JudgeProfile.Propositions, dealDocID)
			// Save that judges
			err := UpdateUserDB(ctx, j.ID.Hex(), j, s.userTable)
			if err != nil {
				fmt.Println("[LOG]:", "Failed to update judge "+j.ID.Hex()+" propositions: ", err)
				return err
			}
		}
	}
	return nil
}

func (s *service) JudgeAccept(ctx context.Context, judgeID, dealDocID string) error {
	// Get judge
	judge, err := GetUserByIDDB(ctx, judgeID, s.userTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get user from DB, err: ", err)
		return err
	}
	if judge == nil {
		err := status.Errorf(codes.InvalidArgument, "Judge with id "+judge.ID.Hex()+" doesn't exist")
		fmt.Println("[LOG]:", err.Error())
		return err
	}
	if judge.IsJudge == false {
		err := status.Errorf(codes.InvalidArgument, "User with id "+judge.ID.Hex()+" is not judge")
		fmt.Println("[LOG]:", err.Error())
		return err
	}
	if judge.JudgeProfile == nil {
		err := status.Errorf(codes.Internal, "Judge with id "+judge.ID.Hex()+" has invalid data")
		fmt.Println("[LOG]:", err.Error())
		return err
	}
	// Check deal status, because someone could already take it
	dealDoc, err := GetDealDocByIdDB(ctx, dealDocID, s.dealDocTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get deal document from DB, err: ", err)
		return err
	}
	dealStatus, err := (*dealDoc).getStatus()
	if err != nil {
		return err
	}
	if dealStatus == "ACCEPTED_BY_USERS" {
		// If deal still waiting for judge
		// Move deal from judge [Propositions] to [Participations]
		propositionAccepted := false
		for i, p := range judge.JudgeProfile.Propositions {
			if p == dealDocID {
				propositionAccepted = true
				judge.JudgeProfile.Propositions = append(judge.JudgeProfile.Propositions[:i], judge.JudgeProfile.Propositions[i+1:]...)
				judge.JudgeProfile.Participatings = append(judge.JudgeProfile.Participatings, dealDocID)
				err := UpdateUserDB(ctx, judge.ID.Hex(), judge, s.userTable)
				if err != nil {
					fmt.Println("[LOG]:", "Failed to update judge "+judge.ID.Hex()+" propositions: ", err)
					return err
				}
				// All participants accepted, deal is ready to wait for resolve
				err = JudgeAcceptDeal(ctx, judgeID, dealDocID, s.dealDocTable)
				if err != nil {
					fmt.Println("[LOG]:", "Failed to update deal "+dealDocID+" status: ", err)
					return err
				}
				// Update deal status and call watcher
				err = s.SendDealToWatcher(ctx, dealDoc)
				if err != nil {
					fmt.Println("[LOG]:", "Failed to send deal "+dealDoc.ID.Hex()+" to watcher service: ", err)
					return err
				}
				break
			}
		}
		if !propositionAccepted {
			err := status.Errorf(codes.Internal, "Judge with id "+judge.ID.Hex()+" doesn't have deal with id "+dealDocID+" in propositions")
			fmt.Println("[LOG]:", err.Error())
			return err
		}
		return nil
	}
	// In case of another status
	// If deal already taken by another judge
	for i, p := range judge.JudgeProfile.Propositions {
		// Remove dealID from propositions
		if p == dealDocID {
			judge.JudgeProfile.Propositions = append(judge.JudgeProfile.Propositions[:i], judge.JudgeProfile.Propositions[i+1:]...)
			err := UpdateUserDB(ctx, judge.ID.Hex(), judge, s.userTable)
			if err != nil {
				fmt.Println("[LOG]:", "Failed to update judge "+judge.ID.Hex()+" propositions: ", err)
				return err
			}
			break
		}
	}

	return nil
}

func (s *service) SendDealToWatcher(ctx context.Context, dealDoc *DealDocumentDB) error {
	dealID := dealDoc.ID.Hex()
	currentPact, err := (*dealDoc).getCurrentPact()
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get deal document current pact: ", err)
		return err
	}
	// Send deal to watcher
	_, err = s.watcherSvcClient.HoldAndWatch(ctx, &pb.HoldAndWatchReq{
		ReqHdr: &pb.ReqHdr{
			Tid: "1234",
		},
		DealId:  dealID,
		Timeout: currentPact.Timeout,
	})
	if err != nil {
		fmt.Println("[LOG]:", "Failed to watch new deal: ", err)
		return err
	}
	// Update user deal status
	err = TellUserDealStarted(ctx, *dealDoc, s.userTable)
	if err != nil {
		fmt.Printf("[LOG]: Failed to update user statuses to [PARTICIPATING] in deal %s: %s\n", dealDoc.ID, err)
		return err
	}
	return nil
}

// DealTimeout can be called only by watcherSvc that watch a timer for deal timeout
func (s *service) DealTimeout(ctx context.Context, dealDocID string) error {
	fmt.Println("[LOG]:", "Inside DealTimeout, docID: ", dealDocID)
	dealDoc, err := GetDealDocByIdDB(ctx, dealDocID, s.dealDocTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get deal document current pact: ", err)
		return err
	}
	if dealDoc == nil {
		err = fmt.Errorf("[LOG]:", "No deal with id %s, error: %v", dealDocID, err)
		return err
	}
	// Check if winner chosen and set winner status or expiration
	dealStatus, err := dealDoc.getStatus()
	pact, err := dealDoc.getCurrentPact()
	blueParticipants := pact.Blue.Participants
	redParticipants := pact.Red.Participants

	// Notify users about deal result
	if dealStatus == "WINNER_SET" {
		blueStatus := "losed"
		redStatus := "won"
		if dealDoc.Winner == "blue" {
			blueStatus, redStatus = redStatus, blueStatus
		}
		for _, rP := range redParticipants {
			err = notifyParticipantAboutResult(ctx, dealDoc.ID.Hex(), rP, redStatus, s.userTable)
			if err != nil {
				return fmt.Errorf("Failed to notify user about deal result: %v", err)
			}
		}
		for _, bP := range blueParticipants {
			err = notifyParticipantAboutResult(ctx, dealDoc.ID.Hex(), bP, blueStatus, s.userTable)
			if err != nil {
				return fmt.Errorf("Failed to notify user about deal result: %v", err)
			}
		}
	} else {
		// Judge hasn't set the winner so we will set deal result as expired
		participants := append(blueParticipants, redParticipants...)
		for _, p := range participants {
			err = notifyParticipantAboutResult(ctx, dealDoc.ID.Hex(), p, "EXPIRED", s.userTable)
			if err != nil {
				return fmt.Errorf("Failed to notify user about deal result: %v", err)
			}
		}
	}
	err = UpdateDealStatus(ctx, dealDocID, "TIME_OUT", s.dealDocTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to update deal status: ", err)
	}
	return err
}

func notifyParticipantAboutResult(ctx context.Context, dealDocID string, participant ParticipantDB, status string, userTable *mongo.Collection) error {
	fmt.Println("{DEBUG}:", "Inside notifyParticipantAboutResult")
	fmt.Println("{DEBUG}:", "Update status of "+participant.ID+" to status "+status)
	// Get user
	user, err := GetUserByIDDB(ctx, participant.ID, userTable)
	if err != nil {
		fmt.Println("[LOG]", "Failed to get user from DB to set winner status")
	}
	// Check if user participating in the deal
	partDealIndex := -1
	for i, participatedDeal := range user.Participating {
		if participatedDeal == dealDocID {
			partDealIndex = i
			break
		}
	}
	if partDealIndex == -1 {
		fmt.Println("{DEBUG}:", "User "+participant.ID+" doesn't participate in deal "+dealDocID)
		return errors.New("User " + participant.ID + " doesn't participate in deal " + dealDocID)
	}
	// Move from participating to deal_results
	user.Participating = append(user.Participating[:partDealIndex], user.Participating[partDealIndex+1:]...)
	user.DealResults = append(user.DealResults, dealDocID)
	err = UpdateUserDB(ctx, participant.ID, user, userTable)
	if err != nil {
		return fmt.Errorf("Failed to notify user %s about of %s result of deal %s", participant.ID, status, dealDocID)
	}
	return err
}

// JudgeDecide make a decision who won that deal (red if redWon is true)
func (s *service) JudgeDecide(ctx context.Context, judgeID, dealDocID, winner string) error {
	fmt.Println("Inputs: [judgeID]: ", judgeID, "[dealDocID]: ", dealDocID, "[winner]: ", winner)
	dealDoc, err := GetDealDocByIdDB(ctx, dealDocID, s.dealDocTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get deal document current pact: ", err)
		return err
	}
	if dealDoc == nil {
		return fmt.Errorf("No deal with id %s, error: %v", dealDocID, err)
	}
	// Get judge profile
	judge, err := GetUserByIDDB(ctx, judgeID, s.userTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get judge: ", err)
		return err
	}
	if judge == nil {
		return fmt.Errorf("No judge with id %s, error: %v", judgeID, err)
	}
	// Check whether judge participate in the deal
	participatingInDeal := false
	for _, participatingDealID := range judge.JudgeProfile.Participatings {
		if participatingDealID == dealDocID {
			participatingInDeal = true
			break
		}
	}
	if !participatingInDeal {
		return fmt.Errorf("Judge %s doesn't participate in deal %s", judgeID, dealDocID)
	}
	err = SetDealWinner(ctx, dealDocID, winner, s.dealDocTable, s.userTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to set deal winner: ", err)
	}
	err = MakeDecision(ctx, judge, dealDocID, winner, s.userTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to update judge decisions stats: ", err)
	}
	return err
}

func (s *service) JoinBlame(ctx context.Context, userID, blameID string) error {
	userDB, err := GetUserByIDDB(ctx, userID, s.userTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get user %s from DB, err: ", userID, err)
		return err
	}
	if !userDB.IsJudge {
		return fmt.Errorf("User %s can't join blame %s because he is not a judge", userID, blameID)
	}
	for _, pD := range userDB.Participating {
		if pD == blameID {
			return fmt.Errorf("User %s can't join blame %s because he is already participating", userID, blameID)
		}
	}
	userJustice, err := userDB.getJustice(ctx, s.dealDocTable)
	if err != nil {
		return fmt.Errorf("Failed to get user %s justice, err: %v", userID, err)
	}
	// Gotcha, hacker
	if userJustice < 0 {
		return fmt.Errorf("User %s can't join blame %s because his justice level is toxic", userID, blameID)
	}
	if userJustice == 0 {
		return fmt.Errorf("User %s can't join blame %s because his justice level is useless", userID, blameID)
	}
	// Update user deals for participation and judge
	blameDoc, err := GetDealDocByIdDB(ctx, blameID, s.dealDocTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get blame %s document, err: ", blameID, err)
		return err
	}
	if blameDoc.Completed {
		return fmt.Errorf("User %s can't join blame %s because it's already completed", userID, blameID)
	}
	for i, p := range blameDoc.Pacts {
		if p.Version == blameDoc.FinalVersion {
			// We need to update this pact
			blameDoc.Pacts[i].Red.Participants = append(blameDoc.Pacts[i].Red.Participants, ParticipantDB{
				ID:       userID,
				Accepted: true,
			})
			// Add new judge and increase justice
			blameDoc.Judge.Participants = append(blameDoc.Judge.Participants, ParticipantDB{
				ID:       userID,
				Accepted: true,
			})
			blameDoc.JusticeCount += userJustice
			break
		}
	}
	// Save deal document
	err = UpdateDeal(ctx, *blameDoc, s.dealDocTable)
	if err != nil {
		return fmt.Errorf("User %s can't join deal because his justice level is useless")
	}
	// Update user
	userDB.Participating = append(userDB.Participating, blameID)
	userDB.JudgeProfile.Decisions = append(userDB.JudgeProfile.Decisions, Decision{
		DealID: blameID,
		Winner: "Blame",
		When:   time.Now().String(),
	})

	err = UpdateUserDB(ctx, userID, userDB, s.userTable)
	if err != nil {
		return fmt.Errorf("Failed to update user %s, err: %v", userID, err)
	}
	return nil
}

func (s *service) ActivateBlame(ctx context.Context, judgeID, blameID string) error {
	userDB, err := GetUserByIDDB(ctx, judgeID, s.userTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get user %s from DB, err: ", judgeID, err)
		return err
	}
	if !userDB.IsJudge {
		return fmt.Errorf("User %s can't join blame %s because he is not a judge", judgeID, blameID)
	}
	participation := false
	for _, pD := range userDB.Participating {
		if pD == blameID {
			participation = true
			break
		}
	}
	if !participation {
		return fmt.Errorf("User %s can't activate blame %s because he doesn't participate in it", judgeID, blameID)
	}
	// Ok, user can activate blame, let's check whether blame is possible to activate with current justiceCount
	blameDoc, err := GetDealDocByIdDB(ctx, blameID, s.dealDocTable)
	if err != nil {
		return fmt.Errorf("Failed to get blame %s document, err: %v", blameID, err)
	}
	if blameDoc.Completed {
		return fmt.Errorf("User %s can't activate blame %s because it's already activated", judgeID, blameID)
	}
	var blamedDealID string
	for _, p := range blameDoc.Pacts {
		if p.Version == blameDoc.FinalVersion {
			blamedDealID = p.Blue.Participants[0].ID
		}
	}
	blamedDealDoc, err := GetDealDocByIdDB(ctx, blamedDealID, s.dealDocTable)
	if err != nil {
		return fmt.Errorf("Failed to get blamed deal document %s document, err: %v", blamedDealID, err)
	}
	if blamedDealDoc.JusticeCount > blameDoc.JusticeCount {
		return fmt.Errorf("User %s can't activate blame %s because bale justice count %d is not enough, %d needed", judgeID, blameID, blameDoc.JusticeCount, blamedDealDoc.JusticeCount)
	}

	blameDoc.Completed = true
	blameDoc.Blamed = "No"
	err = UpdateDeal(ctx, *blameDoc, s.dealDocTable)
	if err != nil {
		return fmt.Errorf("Failed to activate blamed deal document %s, err: %v", blamedDealID, err)
	}
	// Reverse status of blamed deals (that can be chain of documents like BLAME -> BLAME -> ... -> COMMON)
	err = s.blameDeal(ctx, *blamedDealDoc)
	if err != nil {
		return fmt.Errorf("Failed blame chain of documents starting from document %s, err: %v", blamedDealDoc.ID.Hex(), err)
	}
	return nil
}

func (s *service) blameDeal(ctx context.Context, deal DealDocumentDB) error {
	// If you want to blame simple dealm=, just chnage status and uodate it
	switch deal.Type {
	case "COMMON":
		// We reverse, so it the deal was blamed, we have to reverse it (by blaming the blame)
		if deal.Blamed == "Yes" {
			deal.Blamed = "No"
		}
		if deal.Blamed == "No" {
			deal.Blamed = "Yes"
		}
		return UpdateDeal(ctx, deal, s.dealDocTable)
	// To bale the "BLAME" you have to do that recursively
	case "BLAME":
		if deal.Blamed == "Yes" {
			deal.Blamed = "No"
		}
		if deal.Blamed == "Yes" {
			deal.Blamed = "No"
		}
		err := UpdateDeal(ctx, deal, s.dealDocTable)
		if err != nil {
			fmt.Errorf("Failed to update deal document %s, err: %v", deal.ID.Hex(), err)
		}
		var blamedDealID string
		for _, p := range deal.Pacts {
			if p.Version == deal.FinalVersion {
				blamedDealID = p.Blue.Participants[0].ID
			}
		}
		blamedDealDoc, err := GetDealDocByIdDB(ctx, blamedDealID, s.dealDocTable)
		if err != nil {
			return fmt.Errorf("Failed to get blamed deal document %s, err: %v", blamedDealID, err)
		}
		return s.blameDeal(ctx, *blamedDealDoc)
	default:
		return fmt.Errorf("Can't blame deal %s, with unknown type: %v", deal.ID.Hex(), deal.Type)
	}
}
