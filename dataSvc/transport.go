//
// Copyright 2019
//
// @author: Denys Nahurnyi
// @email:  dnahurnyi@gmail.com
// ---------------------------------------------------------------------------
package dataSvc

import (
	"context"

	grpcutils "github.com/DenysNahurnyi/deal/common/grpc"
	pb "github.com/DenysNahurnyi/deal/pb/generated/pb"
	"github.com/go-kit/kit/log"
	grpctransport "github.com/go-kit/kit/transport/grpc"
)

type grpcServer struct {
	createUser grpctransport.Handler
	getUser    grpctransport.Handler
	readiness  grpctransport.Handler
	deleteUser grpctransport.Handler
	updateUser grpctransport.Handler
}

func NewGRPCServer(svc Service, logger log.Logger) pb.DataServiceServer {
	options := []grpctransport.ServerOption{
		grpctransport.ServerErrorLogger(logger),
	}
	return &grpcServer{
		createUser: grpctransport.NewServer(
			makeCreateUserEndpoint(svc),
			decodeCreateUserReq,
			encodeCreateUserResp,
			options...),
		getUser: grpctransport.NewServer(
			makeGetUserEndpoint(svc),
			decodeGetUserReq,
			encodeGetUserResp,
			append(options, grpctransport.ServerBefore(
				grpcutils.ParseCookies(),
				grpcutils.ParseHeader(grpcutils.GRPCAUTHORIZATIONHEADER),
				grpcutils.VerifyToken(svc.GetPubKey())))...,
		),
		deleteUser: grpctransport.NewServer(
			makeDeleteUserEndpoint(svc),
			decodeDeleteUserReq,
			encodeDeleteUserResp,
			append(options, grpctransport.ServerBefore(
				grpcutils.ParseCookies(),
				grpcutils.ParseHeader(grpcutils.GRPCAUTHORIZATIONHEADER),
				grpcutils.VerifyToken(svc.GetPubKey())))...,
		),
		updateUser: grpctransport.NewServer(
			makeUpdateUserEndpoint(svc),
			decodeUpdateUserReq,
			encodeUpdateUserResp,
			append(options, grpctransport.ServerBefore(
				grpcutils.ParseCookies(),
				grpcutils.ParseHeader(grpcutils.GRPCAUTHORIZATIONHEADER),
				grpcutils.VerifyToken(svc.GetPubKey())))...,
		),
		// getTenantMgr: grpctransport.NewServer(
		// 	makeGetTenantEndpoint(svc),
		// 	decodeGetMgrEndpointReq,
		// 	encodeGetMgrEndpointResp,
		// 	append(options, grpctransport.ServerBefore(
		// 		grpcutils.ParseCookies(),
		// 		grpcutils.ParseHeader(grpcutils.GRPCAUTHORIZATIONHEADER),
		// 		grpcutils.VerifyToken(svc.GetDBIf(), logger)))...),
	}
}

func (s *grpcServer) UpdateUser(ctx context.Context, req *pb.UpdateUserReq) (*pb.UpdateUserResp, error) {
	_, resp, err := s.updateUser.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.UpdateUserResp), nil
}

func decodeUpdateUserReq(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.UpdateUserReq)
	return req, nil
}

func encodeUpdateUserResp(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(pb.UpdateUserResp)
	return &resp, nil
}

func (s *grpcServer) DeleteUser(ctx context.Context, req *pb.DeleteUserReq) (*pb.DeleteUserResp, error) {
	_, resp, err := s.deleteUser.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.DeleteUserResp), nil
}

func decodeDeleteUserReq(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.DeleteUserReq)
	return req, nil
}

func encodeDeleteUserResp(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(pb.DeleteUserResp)
	return &resp, nil
}

func (s *grpcServer) CreateUser(ctx context.Context, req *pb.CreateUserReq) (*pb.CreateUserResp, error) {
	_, resp, err := s.createUser.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.CreateUserResp), nil
}

func decodeCreateUserReq(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.CreateUserReq)
	return req, nil
}

func encodeCreateUserResp(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(pb.CreateUserResp)
	return &resp, nil
}

func (s *grpcServer) GetUser(ctx context.Context, req *pb.GetUserReq) (*pb.GetUserResp, error) {
	_, resp, err := s.getUser.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.GetUserResp), nil
}

func decodeGetUserReq(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.GetUserReq)
	return req, nil
}

func encodeGetUserResp(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(pb.GetUserResp)
	return &resp, nil
}
