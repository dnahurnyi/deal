//
// Copyright 2018 Orkus, Inc
// All Rights Reserved.
//
// @author: Denys Nahurnyi, Orkus, Inc.
// @email:  denys.nahurnyi@Blackthorn-vision.com
// ---------------------------------------------------------------------------
package dataMgr

import (
	"context"

	pb "github.com/DenysNahurnyi/deal/pb"
	"github.com/go-kit/kit/log"
	grpctransport "github.com/go-kit/kit/transport/grpc"
)

type grpcServer struct {
	createUser grpctransport.Handler
}

func NewGRPCServer(svc Service, logger log.Logger) pb.TenantMgrServer {
	options := []grpctransport.ServerOption{
		grpctransport.ServerErrorLogger(logger),
	}
	return &grpcServer{
		createUser: grpctransport.NewServer(
			makeCreateUserEndpoint(svc),
			decodeCreateUserReq,
			encodeCreateUserResp,
			options...),
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

func (s *grpcServer) CreateUserEndpoint(ctx context.Context, req *pb.CreateUserReq) (*pb.CreateUserResp, error) {
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
