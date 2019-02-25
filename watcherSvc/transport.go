//
// Copyright 2019
//
// @author: Denys Nahurnyi
// @email:  dnahurnyi@gmail.com
// ---------------------------------------------------------------------------
package watcherSvc

import (
	"context"

	pb "github.com/DenysNahurnyi/deal/pb/generated/pb"
	"github.com/go-kit/kit/log"
	grpctransport "github.com/go-kit/kit/transport/grpc"
)

type grpcServer struct {
	holdAndWatch grpctransport.Handler
}

func NewGRPCServer(svc Service, logger log.Logger) pb.WatcherServiceServer {
	options := []grpctransport.ServerOption{
		grpctransport.ServerErrorLogger(logger),
	}
	return &grpcServer{
		holdAndWatch: grpctransport.NewServer(
			makeHoldAndWatchEndpoint(svc),
			decodeHoldAndWatchReq,
			encodeHoldAndWatchResp,
			options...),
	}
}

func (s *grpcServer) HoldAndWatch(ctx context.Context, req *pb.HoldAndWatchReq) (*pb.HoldAndWatchResp, error) {
	_, resp, err := s.holdAndWatch.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.HoldAndWatchResp), nil
}

func decodeHoldAndWatchReq(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.HoldAndWatchReq)
	return req, nil
}

func encodeHoldAndWatchResp(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(pb.HoldAndWatchResp)
	return &resp, nil
}
