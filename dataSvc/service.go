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
)

type Service interface {
	CreateUser(ctx context.Context, user *pb.User) (string, error)
	GetUser(ctx context.Context) (*pb.User, error)
	DeleteUser(ctx context.Context) (*pb.User, error)
	UpdateUser(ctx context.Context, user *pb.User) (*pb.User, error)
	CreateDealDocument(ctx context.Context, dealDocument *pb.Pact) (string, error)
	GetDealDocument(ctx context.Context, dealDocumentID string) (*pb.DealDocument, error)
	OfferDealDocument(ctx context.Context, dealDocId, username string) error
	AcceptDealDocument(ctx context.Context, dealDocId string, side pb.SideType) error
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

func (s *service) CreateUser(ctx context.Context, userReq *pb.User) (string, error) {
	userGet, _, err := GetUserByUsernameDB(ctx, userReq.GetUsername(), s.userTable)
	if err != nil {
		fmt.Println("Failed to get user from DB")
		return "", err
	}
	if len(userGet.GetUsername()) > 0 {
		fmt.Println("[WARNING] user already exist")
		return "", errors.New("User already exist")
	}
	userID, err := CreateUserDB(ctx, UserDB{
		Name:     userReq.GetName(),
		Surname:  userReq.GetSurname(),
		Username: userReq.GetUsername(),
	}, s.userTable)
	return userID, err
}

func (s *service) GetUser(ctx context.Context) (*pb.User, error) {
	userID, err := grpcutils.GetUserIDFromJWT(ctx)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get user id from token, err: ", err)
		return nil, err
	}

	return GetUserByIdDBConvert(ctx, userID, s.userTable)
}

func (s *service) DeleteUser(ctx context.Context) (*pb.User, error) {
	userID, err := grpcutils.GetUserIDFromJWT(ctx)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get user id from token, err: ", err)
		return nil, err
	}

	user, err := DeleteUserByIdDB(ctx, userID, s.userTable)
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

func (s *service) UpdateUser(ctx context.Context, user *pb.User) (*pb.User, error) {
	userID, err := grpcutils.GetUserIDFromJWT(ctx)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get user id from token, err: ", err)
		return nil, err
	}

	userExist, err := GetUserByIdDBConvert(ctx, userID, s.userTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get user, err: ", err)
		return nil, err
	}
	if len(userExist.GetUsername()) == 0 {
		fmt.Println("[WARNING] user doesn't exist")
		return nil, errors.New("User doesn't exist")
	}
	err = UpdateUserDB(ctx, userID, &UserDB{
		Name:     user.GetName(),
		Surname:  user.GetSurname(),
		Username: user.GetUsername(),
	}, s.userTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to update user in data service, err: ", err)
		return nil, err
	}
	return user, nil
}

func (s *service) CreateDealDocument(ctx context.Context, dealDocument *pb.Pact) (string, error) {
	userID, err := grpcutils.GetUserIDFromJWT(ctx)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get user id from token, err: ", err)
		return "", err
	}
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
	user, err := GetUserByIdDBConvert(ctx, userID, s.userTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get user from DB, err: ", err)
		return "", err
	}
	err = UpdateUserDB(ctx, userID, &UserDB{
		DealDocs: append(user.GetDealDocs(), dealDocID),
	}, s.userTable)
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

// OfferDealDocument offer another user deal document
func (s *service) OfferDealDocument(ctx context.Context, dealDocID, username string) error {
	_, err := grpcutils.GetUserIDFromJWT(ctx)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get user id from token, err: ", err)
		return err
	}
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
	err = UpdateUserDB(ctx, offeredUserID, &UserDB{
		Offerings: append(offeredUser.GetOfferings(), dealDocID),
	}, s.userTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to user in DB, err: ", err)
		return err
	}
	return OfferDealDocDB(ctx, dealDocID, offeredUserID, pb.SideType_BLUE, s.dealDocTable)
}

func (s *service) GetDealDocument(ctx context.Context, dealDocumentID string) (*pb.DealDocument, error) {
	_, err := grpcutils.GetUserIDFromJWT(ctx)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get user id from token, err: ", err)
		return nil, err
	}
	dealDoc, err := GetDealDocByIdDBConvert(ctx, dealDocumentID, s.dealDocTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get deal document, err: ", err)
		return nil, err
	}
	return dealDoc, err
}

// AcceptDealDocument accepts `dealID` deal for `userID` user
func (s *service) AcceptDealDocument(ctx context.Context, dealDocID string, side pb.SideType) error {
	userID, err := grpcutils.GetUserIDFromJWT(ctx)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get user id from token, err: ", err)
		return err
	}
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
	isDealAccepted, err := CheckToWatchDeal(ctx, dealDocID, s.dealDocTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to check wheter deal accepted by every participant: ", err)
		return err
	}
	if isDealAccepted {
		// Update deal status
		err = UpdateDealStatus(ctx, dealDocID, "ACCEPTED", s.dealDocTable)
		if err != nil {
			fmt.Println("[LOG]:", "Failed to deal status: ", err)
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
