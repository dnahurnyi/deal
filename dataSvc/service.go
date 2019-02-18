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
	GetPubKey() *rsa.PublicKey
}

type service struct {
	envType       string
	mongoClient   *mongo.Client
	userTable     *mongo.Collection
	dealDocTable  *mongo.Collection
	authSvcClient pb.AuthServiceClient
	uKey          *rsa.PublicKey
}

func NewService(logger log.Logger, mgc *mongo.Client, authSvcClient *pb.AuthServiceClient) (Service, error) {
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
	return &service{
		envType:       "test",
		mongoClient:   mgc,
		userTable:     userTable,
		dealDocTable:  dealDocTable,
		authSvcClient: authSvcClientValue,
		uKey:          uKey,
	}, nil
}

func (s *service) CreateUser(ctx context.Context, userReq *pb.User) (string, error) {
	userGet, err := GetUserByUsernameDB(ctx, userReq.GetUsername(), s.userTable)
	if err != nil {
		fmt.Println("Failed to get user from DB")
		return "", err
	}
	if len(userGet.GetUsername()) > 0 {
		fmt.Println("[WARNING] user already exist")
		return "", errors.New("User already exist")
	}

	userID, err := CreateUserDB(ctx, &UserDB{
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

	return GetUserByIdDB(ctx, userID, s.userTable)
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

	userExist, err := GetUserByIdDB(ctx, userID, s.userTable)
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
	dealDocumentDB, err := createInitDealDocument(userID, dealDocument.GetContent(), dealDocument.GetTimeout().String())
	if err != nil {
		fmt.Println("[LOG]:", "Failed to create deal document, err: ", err)
		return "", err
	}
	dealDocID, err := CreateDealDocumentDB(ctx, dealDocumentDB, s.dealDocTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to create deal document, err: ", err)
		return "", err
	}
	user, err := GetUserByIdDB(ctx, userID, s.userTable)
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
		Version: "initial(#1)",
		Timeout: timeout,
	}
	return DealDocumentDB{
		Pacts:        []PactDB{firstPact},
		FinalVersion: firstPact.Version,
	}, nil
}
