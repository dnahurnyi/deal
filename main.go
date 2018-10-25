package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	pb "github.com/orkusinc/api/common/pb/generated"
	"github.com/orkusinc/api/common/utils"
	"github.com/orkusinc/api/tenantMgr"
)

func main() {
	grpcPort := ":8094"
	errChan := make(chan error)

	fmt.Println("Hello world")

	go func() {
		listener, err := net.Listen("tcp", grpcPort)
		if err != nil {
			errChan <- err
			return
		}
		handler := tenantMgr.NewGRPCServer(svc, logger)
		gRPCServer := grpcutils.NewServer()
		pb.RegisterTenantMgrServer(gRPCServer, handler)
		errChan <- gRPCServer.Serve(listener)
	}()

	go func() {
		opts := grpcutils.OptsGrpcGw()
		mux := runtime.NewServeMux()
		err := pb.RegisterTenantMgrHandlerFromEndpoint(ctx, mux, grpcPort, opts)
		if err != nil {
			errChan <- err
			return
		}
		errChan <- http.ListenAndServe(":8093", mux)
	}()
	go func() {
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
		logger.Log(utils.ForLogs("Listening for signals"))
		errChan <- logger.Log(utils.ForLogs(<-signalCh))
	}()
	logger.Log(utils.ForLogs(<-errChan))
}
