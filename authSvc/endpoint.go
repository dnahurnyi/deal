//
// Copyright 2019
//
// @author: Denys Nahurnyi
// @email:  dnahurnyi@gmail.com
// ---------------------------------------------------------------------------
package authSvc

import (
	"context"

	pb "github.com/DenysNahurnyi/deal/pb/generated/pb"
	"github.com/go-kit/kit/endpoint"
)

func makeLoginEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.LoginReq)
		tid := req.ReqHdr.Tid
		token, err := svc.Login(ctx, req.GetUsername(), req.GetPassword())

		return pb.LoginResp{
			RespHdr: &pb.RespHdr{Tid: tid, ReqTid: tid},
			Token:   token,
		}, err
	}
}

func makeSignUpEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.SignUpReq)
		tid := req.ReqHdr.Tid
		userId, err := svc.SignUp(ctx, req.GetUserReq(), req.GetPassword())

		return pb.SignUpResp{
			RespHdr: &pb.RespHdr{Tid: tid, ReqTid: tid},
			UserId:  userId,
		}, err
	}
}
