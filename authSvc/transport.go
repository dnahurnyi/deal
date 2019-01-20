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
	"github.com/go-kit/kit/log"
	grpctransport "github.com/go-kit/kit/transport/grpc"
)

type grpcServer struct {
	login grpctransport.Handler
}

func NewGRPCServer(svc Service, logger log.Logger) pb.AuthServiceServer {
	options := []grpctransport.ServerOption{
		grpctransport.ServerErrorLogger(logger),
	}
	return &grpcServer{
		login: grpctransport.NewServer(
			makeLoginEndpoint(svc),
			decodeLoginReq,
			encodeLoginResp,
			options...),
	}
}

func (s *grpcServer) Login(ctx context.Context, req *pb.LoginReq) (*pb.LoginResp, error) {
	_, resp, err := s.login.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.LoginResp), nil
}

func decodeLoginReq(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.LoginReq)
	return req, nil
}

func encodeLoginResp(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(pb.LoginResp)
	return &resp, nil
}
