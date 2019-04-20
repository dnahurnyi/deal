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
	GetDealDocument(ctx context.Context, dealDocumentID string) (*pb.DealDocument, error)
	OfferDealDocument(ctx context.Context, dealDocId, username string, toJudge bool) error
	AcceptDealDocument(ctx context.Context, userID, dealDocId string, side pb.SideType) error
	OfferJudges(ctx context.Context, dealDocId string) error
	JudgeAccept(ctx context.Context, judgeID, dealDocId string) error
	GetPubKey() *rsa.PublicKey
}

type service struct {
	envType          string
	mongoClient      *mongo.Client
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
		mongoClient:      mgc,
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

func (s *service) CreateDealDocument(ctx context.Context, userID string, dealDocument *pb.Pact) (string, error) {
	dealDocumentDB, err := createInitDealDocument(userID, dealDocument.GetContent(), dealDocument.GetTimeout())
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

func createInitDealDocument(redUserID, content, timeout string) (DealDocumentDB, error) {
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
		Pacts:        []PactDB{firstPact},
		FinalVersion: firstPact.Version,
		Status:       "INITIAL DEAL STAGE",
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
		// Send deal to watcher
		_, err := s.watcherSvcClient.HoldAndWatch(ctx, &pb.HoldAndWatchReq{
			ReqHdr: &pb.ReqHdr{
				Tid: "1234",
			},
			DealId: dealDocID,
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
	if dealDoc.Status == "ACCEPTED_BY_USERS" {
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
