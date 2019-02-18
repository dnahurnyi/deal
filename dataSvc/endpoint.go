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

	pb "github.com/DenysNahurnyi/deal/pb/generated/pb"
	"github.com/go-kit/kit/endpoint"
)

func makeCreateUserEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.CreateUserReq)
		tid := req.ReqHdr.Tid
		userId, err := svc.CreateUser(ctx, req.GetUser())

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
		user, err := svc.GetUser(ctx)

		return pb.GetUserResp{
			RespHdr: &pb.RespHdr{Tid: tid, ReqTid: tid},
			User:    user,
		}, err
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
		user, err := svc.DeleteUser(ctx)

		return pb.DeleteUserResp{
			RespHdr: &pb.RespHdr{Tid: tid, ReqTid: tid},
			User:    user,
		}, err
	}
}

func makeUpdateUserEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.UpdateUserReq)
		tid := req.ReqHdr.Tid
		user, err := svc.UpdateUser(ctx, req.GetUser())

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
		dealDocumentID, err := svc.CreateDealDocument(ctx, req.GetDealDocument())

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
		err := svc.OfferDealDocument(ctx, req.GetDealDocId(), req.GetUsername())

		return pb.OfferDealDocumentResp{
			RespHdr: &pb.RespHdr{Tid: tid, ReqTid: tid},
		}, err
	}
}
