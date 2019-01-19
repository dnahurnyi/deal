//
// Copyright 2019
//
// @author: Denys Nahurnyi
// @email:  dnahurnyi@gmail.com
// ---------------------------------------------------------------------------
package dataSvc

import (
	"context"

	pb "github.com/DenysNahurnyi/deal/pb/generated/pb"
	"github.com/go-kit/kit/endpoint"
)

func makeCreateUserEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.CreateUserReq)
		tid := req.ReqHdr.Tid
		svc.CreateUser(ctx, req.GetUser())

		return pb.CreateUserResp{
			RespHdr: &pb.RespHdr{Tid: tid, ReqTid: tid},
			// Status:  "tenant",
		}, nil
	}
}

func makeGetUserEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.GetUserReq)
		tid := req.ReqHdr.Tid
		svc.GetUser(ctx, req.GetUserId())

		return pb.GetUserResp{
			RespHdr: &pb.RespHdr{Tid: tid, ReqTid: tid},
			User:    nil,
		}, nil
	}
}
