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
	login            grpctransport.Handler
	signUp           grpctransport.Handler
	deleteUser       grpctransport.Handler
	getCheckTokenKey grpctransport.Handler
	existenceCheck   grpctransport.Handler
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
		signUp: grpctransport.NewServer(
			makeSignUpEndpoint(svc),
			decodeSignUpReq,
			encodeSignUpResp,
			options...),
		getCheckTokenKey: grpctransport.NewServer(
			makeGetCheckTokenKeyEndpoint(svc),
			decodeEmptyReq,
			encodeCheckTokenKeyResp,
			options...),
		deleteUser: grpctransport.NewServer(
			makeDeleteSecureUserReqEndpoint(svc),
			decodeDeleteUserReq,
			encodeEmptyResp,
			options...),
		existenceCheck: grpctransport.NewServer(
			makeExistenceCheckEndpoint(svc),
			decodeEmptyReq,
			encodeEmptyResp,
			options...),
	}
}

func (s *grpcServer) ExistenceCheck(ctx context.Context, req *pb.EmptyReq) (*pb.EmptyResp, error) {
	_, resp, err := s.existenceCheck.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.EmptyResp), nil
}

func (s *grpcServer) DeleteUser(ctx context.Context, req *pb.DeleteSecureUserReq) (*pb.EmptyResp, error) {
	_, resp, err := s.deleteUser.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.EmptyResp), nil
}

func decodeDeleteUserReq(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.DeleteSecureUserReq)
	return req, nil
}

func encodeEmptyResp(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(pb.EmptyResp)
	return &resp, nil
}

func (s *grpcServer) GetCheckTokenKey(ctx context.Context, req *pb.EmptyReq) (*pb.CheckTokenKeyResp, error) {
	_, resp, err := s.getCheckTokenKey.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.CheckTokenKeyResp), nil
}

func decodeEmptyReq(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.EmptyReq)
	return req, nil
}

func encodeCheckTokenKeyResp(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(pb.CheckTokenKeyResp)
	return &resp, nil
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

func (s *grpcServer) SignUp(ctx context.Context, req *pb.SignUpReq) (*pb.SignUpResp, error) {
	_, resp, err := s.signUp.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.SignUpResp), nil
}

func decodeSignUpReq(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.SignUpReq)
	return req, nil
}

func encodeSignUpResp(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(pb.SignUpResp)
	return &resp, nil
}
