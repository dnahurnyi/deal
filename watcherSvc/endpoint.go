package watcherSvc

import (
	"context"

	"github.com/DenysNahurnyi/deal/pb/generated/pb"
	"github.com/go-kit/kit/endpoint"
)

func makeHoldAndWatchEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.HoldAndWatchReq)
		tid := "unknown"
		if req.ReqHdr != nil {
			tid = req.ReqHdr.Tid
		}
		err := svc.HoldAndWatch(ctx, req.GetDealId(), req.GetTimeout())

		return pb.HoldAndWatchResp{
			RespHdr: &pb.RespHdr{Tid: tid, ReqTid: tid},
			Status:  "In development",
		}, err
	}
}
