//
// Copyright 2019
//
// @author: Denys Nahurnyi
// @email:  dnahurnyi@gmail.com
// ---------------------------------------------------------------------------
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/DenysNahurnyi/deal/authSvc"
	"github.com/DenysNahurnyi/deal/common/grpc"
	pb "github.com/DenysNahurnyi/deal/pb/generated/pb"
	"github.com/go-kit/kit/log"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/mongodb/mongo-go-driver/mongo"
)

func main() {
	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stdout)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}
	grpcPort := ":8012"
	errChan := make(chan error)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	fmt.Println("Hello world")
	client, err := mongo.Connect(ctx, "mongodb://localhost:27017")
	fmt.Println("Mongo error: ", err)

	svc, _ := authSvc.NewService(logger, client)
	go func() {
		listener, err := net.Listen("tcp", grpcPort)
		if err != nil {
			errChan <- err
			return
		}
		handler := authSvc.NewGRPCServer(svc, logger)
		gRPCServer := grpcutils.NewServer()
		pb.RegisterAuthServiceServer(gRPCServer, handler)

		errChan <- gRPCServer.Serve(listener)
	}()

	go func() {
		fmt.Println("#3")
		opts := grpcutils.OptsGrpcGw()
		mux := runtime.NewServeMux()
		err := pb.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, grpcPort, opts)
		if err != nil {
			errChan <- err
			return
		}
		errChan <- http.ListenAndServe(":8013", mux)
	}()
	go func() {
		fmt.Println("#4")
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
		logger.Log("Listening for signals")
		errChan <- logger.Log(<-signalCh)
	}()
	logger.Log(<-errChan)
}
