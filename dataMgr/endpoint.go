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

	pb "github.com/orkusinc/api/common/pb/generated"
)

func makeCreateUserEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*pb.GetTenantReq)
		tid := req.ReqHdr.Tid
		tenant, err := svc.GetTenant(ctx, req.GetTenantId())

		return pb.GetTenantResp{
			RespHdr: &pb.RespHdr{Tid: tid, ReqTid: tid},
			Tenant:  tenant,
		}, err
	}
}
