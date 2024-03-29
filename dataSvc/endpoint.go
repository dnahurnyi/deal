//
// Copyright 2019
//
// @author: Denys Nahurnyi
// @email:  dnahurnyi@gmail.com
// ---------------------------------------------------------------------------
package dataSvc

import (
	"context"
	"fmt"

	grpcutils "github.com/DenysNahurnyi/deal/common/grpc"
	pb "github.com/DenysNahurnyi/deal/pb/generated/pb"
	"github.com/go-kit/kit/endpoint"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func makeCreateUserEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.CreateUserReq)
		tid := req.ReqHdr.Tid
		userDB, err := ConvertUserToDB(req.GetUser())
		if err != nil {
			return nil, err
		}
		userId, err := svc.CreateUser(ctx, userDB)

		return pb.CreateUserResp{
			RespHdr: &pb.RespHdr{Tid: tid, ReqTid: tid},
			UserId:  userId,
		}, err
	}
}

func makeGetUserEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.GetUserReq)
		tid := req.ReqHdr.Tid
		userID, err := grpcutils.GetUserIDFromJWT(ctx)
		if err != nil {
			fmt.Println("[LOG]:", "Failed to get user id from token, err: ", err)
			return nil, err
		}
		userDB, err := svc.GetUser(ctx, userID)
		if err != nil {
			fmt.Println("[LOG]:", "Failed to get user, err: ", err)
			return nil, err
		}
		user, err := ConvertDBToUser(userDB)
		if err != nil {
			fmt.Println("[LOG]:", "Failed to convert DB user format to response, err: ", err)
			return nil, err
		}
		success, err := userDB.getSuccess(ctx, svc.getDealsTable())
		if err != nil {
			fmt.Println("[LOG]:", "Failed to count user success, err: ", err)
			return nil, err
		}
		user.Success = int64(success)
		fmt.Println("{DEBUG}", "userID: ", userID, " user.JudgeProfile", user.JudgeProfile, " user.IsJudge: ", user.IsJudge)
		if user.JudgeProfile != nil && user.IsJudge {
			justice, err := userDB.getJustice(ctx, svc.getDealsTable())
			fmt.Println("{DEBUG}", "userID: ", userID, " justice, err: ", justice, err)
			if err != nil {
				fmt.Println("[LOG]:", "Failed to count judge justice, err: ", err)
				return nil, err
			}
			user.JudgeProfile.Justice = int64(justice)
		}

		return pb.GetUserResp{
			RespHdr: &pb.RespHdr{Tid: tid, ReqTid: tid},
			User:    user,
		}, nil
	}
}

func makeReadinessEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		fmt.Println("Readiness called")
		return pb.Blank{}, nil
	}
}

func makeDeleteUserEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.DeleteUserReq)
		tid := req.ReqHdr.Tid
		userID, err := grpcutils.GetUserIDFromJWT(ctx)
		if err != nil {
			fmt.Println("[LOG]:", "Failed to get user id from token, err: ", err)
			return nil, err
		}
		userDB, err := svc.DeleteUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		user, err := ConvertDBToUser(userDB)
		if err != nil {
			return nil, err
		}

		return pb.DeleteUserResp{
			RespHdr: &pb.RespHdr{Tid: tid, ReqTid: tid},
			User:    user,
		}, err
	}
}

// This method allows to update only basic properties like name, surname and nickname
func makeUpdateUserEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.UpdateUserReq)
		tid := req.ReqHdr.Tid
		// Validate user input
		userReq := req.GetUser()
		if userReq == nil {
			return nil, status.Errorf(codes.InvalidArgument, "User data musn't be empty")
		}
		// Get user ID
		userID, err := grpcutils.GetUserIDFromJWT(ctx)
		if err != nil {
			fmt.Println("[LOG]:", "Failed to get user id from token, err: ", err)
			return nil, err
		}
		userReq.Id = userID
		// Convert to DB format
		userDB, err := ConvertUserToDB(userReq)
		if err != nil {
			return nil, err
		}

		userDB, err = svc.UpdateUser(ctx, userDB)
		if err != nil {
			return nil, err
		}

		user, err := ConvertDBToUser(userDB)
		if err != nil {
			return nil, err
		}

		return pb.UpdateUserResp{
			RespHdr: &pb.RespHdr{Tid: tid, ReqTid: tid},
			User:    user,
		}, err
	}
}

func makeExistenceCheckEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return pb.EmptyResp{}, nil
	}
}

func makeCreateDealDocumentEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.CreateDealDocumentReq)
		tid := req.ReqHdr.Tid

		userID, err := grpcutils.GetUserIDFromJWT(ctx)
		if err != nil {
			fmt.Println("[LOG]:", "Failed to get user id from token, err: ", err)
			return "", err
		}

		dealDocument := req.GetDealDocument()
		if dealDocument == nil {
			return "", status.Errorf(codes.InvalidArgument, "Deal document musn't be empty")
		}
		if len(dealDocument.Content) == 0 {
			return "", status.Errorf(codes.InvalidArgument, "Deal content musn't be empty")
		}
		if len(dealDocument.Timeout) == 0 {
			return "", status.Errorf(codes.InvalidArgument, "Deal timeout musn't be empty")
		}

		dealDocumentID, err := svc.CreateDealDocument(ctx, userID, req.GetDealDocument())

		return pb.CreateDealDocumentResp{
			RespHdr:        &pb.RespHdr{Tid: tid, ReqTid: tid},
			DealDocumentId: dealDocumentID,
		}, err
	}
}

func makeOfferDealDocumentEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.OfferDealDocumentReq)
		tid := req.ReqHdr.Tid

		_, err := grpcutils.GetUserIDFromJWT(ctx)
		if err != nil {
			fmt.Println("[LOG]:", "Failed to get user id from token, err: ", err)
			return nil, err
		}
		dealDocID := req.GetDealDocId()
		if len(dealDocID) == 0 {
			return nil, status.Errorf(codes.InvalidArgument, "Deal id musn't be empty")
		}
		username := req.GetUsername()
		if len(username) == 0 {
			return nil, status.Errorf(codes.InvalidArgument, "Username musn't be empty")
		}

		err = svc.OfferDealDocument(ctx, dealDocID, username, req.GetToJudge())

		return pb.OfferDealDocumentResp{
			RespHdr: &pb.RespHdr{Tid: tid, ReqTid: tid},
		}, err
	}
}

func makeGetDealDocumentEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.GetDealDocumentReq)
		tid := req.ReqHdr.Tid

		_, err := grpcutils.GetUserIDFromJWT(ctx)
		if err != nil {
			fmt.Println("[LOG]:", "Failed to get user id from token, err: ", err)
			return nil, err
		}

		dealDocument, err := svc.GetDealDocument(ctx, req.GetDealDocumentId())

		return pb.GetDealDocumentResp{
			RespHdr:      &pb.RespHdr{Tid: tid, ReqTid: tid},
			DealDocument: dealDocument,
		}, err
	}
}

func makeAcceptDealDocumentEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.AcceptDealDocumentReq)
		tid := req.ReqHdr.Tid

		userID, err := grpcutils.GetUserIDFromJWT(ctx)
		if err != nil {
			fmt.Println("[LOG]:", "Failed to get user id from token, err: ", err)
			return nil, err
		}
		dealDocID := req.GetDealDocId()
		if len(dealDocID) == 0 {
			fmt.Println("[LOG]:", "Invalid deal document id, err: ", err)
			return nil, err
		}

		err = svc.AcceptDealDocument(ctx, userID, dealDocID, req.GetSideType())

		return pb.AcceptDealDocumentResp{
			RespHdr: &pb.RespHdr{Tid: tid, ReqTid: tid},
		}, err
	}
}

func makeJudgeAcceptDealDocumentEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.JudgeAcceptDealDocumentReq)
		tid := req.ReqHdr.Tid

		userID, err := grpcutils.GetUserIDFromJWT(ctx)
		if err != nil {
			fmt.Println("[LOG]:", "Failed to get user id from token, err: ", err)
			return nil, err
		}
		dealDocID := req.GetDealDocId()
		if len(dealDocID) == 0 {
			fmt.Println("[LOG]:", "Invalid deal document id, err: ", err)
			return nil, err
		}

		err = svc.JudgeAccept(ctx, userID, dealDocID)

		return pb.JudgeAcceptDealDocumentResp{
			RespHdr: &pb.RespHdr{Tid: tid, ReqTid: tid},
		}, err
	}
}

func makeDealTimeoutEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.DealTimeoutReq)
		tid := req.ReqHdr.Tid

		dealDocID := req.GetDealDocumentId()
		if len(dealDocID) == 0 {
			err := fmt.Errorf("Empty deal document id")
			return nil, err
		}

		err := svc.DealTimeout(ctx, dealDocID)

		return pb.DealTimeoutResp{
			RespHdr: &pb.RespHdr{Tid: tid, ReqTid: tid},
		}, err
	}
}

func makeJudgeDecideEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.JudgeDecideReq)
		tid := req.ReqHdr.Tid

		userID, err := grpcutils.GetUserIDFromJWT(ctx)
		if err != nil {
			fmt.Println("[LOG]:", "Failed to get user id from token, err: ", err)
			return nil, err
		}
		dealDocID := req.GetDealDocumentId()
		if len(dealDocID) == 0 {
			return nil, fmt.Errorf("[LOG]:", "Empty deal document id")
		}
		// Using bool here is bad because in case of missing this prop it will automaticly be false
		// XD
		winner := "blue"
		if req.GetRedWon() {
			winner = "red"
		}

		err = svc.JudgeDecide(ctx, userID, dealDocID, winner)

		return pb.JudgeDecideResp{
			RespHdr: &pb.RespHdr{Tid: tid, ReqTid: tid},
		}, err
	}
}

func makeCreateBlameDocumentEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.CreateBlameDocumentReq)
		tid := req.ReqHdr.Tid

		userID, err := grpcutils.GetUserIDFromJWT(ctx)
		if err != nil {
			fmt.Println("[LOG]:", "Failed to get user id from token, err: ", err)
			return "", err
		}

		blamedDealID := req.GetBlamedDealId()
		if len(blamedDealID) == 0 {
			return "", status.Errorf(codes.InvalidArgument, "Blamed deal ID musn't be empty")
		}
		blameReason := req.GetBlameReson()
		if len(blameReason) == 0 {
			return "", status.Errorf(codes.InvalidArgument, "Blame reason musn't be empty")
		}

		blameDocumentID, err := svc.CreateBlameDocument(ctx, userID, blamedDealID, blameReason)

		return pb.CreateBlameDocumentResp{
			RespHdr:         &pb.RespHdr{Tid: tid, ReqTid: tid},
			BlameDocumentId: blameDocumentID,
		}, err
	}
}

func makeJoinBlameEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.JoinBlameReq)
		tid := req.ReqHdr.Tid

		userID, err := grpcutils.GetUserIDFromJWT(ctx)
		if err != nil {
			fmt.Println("[LOG]:", "Failed to get user id from token, err: ", err)
			return "", err
		}

		blameID := req.GetBlameId()
		if len(blameID) == 0 {
			return "", status.Errorf(codes.InvalidArgument, "Blame ID musn't be empty")
		}

		err = svc.JoinBlame(ctx, userID, blameID)

		return pb.JoinBlameResp{
			RespHdr: &pb.RespHdr{Tid: tid, ReqTid: tid},
		}, err
	}
}

func makeActivateBlameEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.ActivateBlameReq)
		tid := req.ReqHdr.Tid

		userID, err := grpcutils.GetUserIDFromJWT(ctx)
		if err != nil {
			fmt.Println("[LOG]:", "Failed to get user id from token, err: ", err)
			return "", err
		}

		blameID := req.GetBlameId()
		if len(blameID) == 0 {
			return "", status.Errorf(codes.InvalidArgument, "Blame ID musn't be empty")
		}

		err = svc.ActivateBlame(ctx, userID, blameID)

		return pb.ActivateBlameResp{
			RespHdr: &pb.RespHdr{Tid: tid, ReqTid: tid},
		}, err
	}
}
