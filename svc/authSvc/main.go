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
	"github.com/DenysNahurnyi/deal/common/utils"
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
	mongoClient, err := mongo.Connect(ctx, "mongodb://n826:qwerty12345@ds055732.mlab.com:55732/travel")
	if err != nil {
		fmt.Println("Error connecting to mongo: ", err)
		return
	}
	dataSvcClient, err := utils.CreateDataSvcClient(logger)
	if err != nil {
		fmt.Println("Error creating client for dataSvc: ", err)
		return
	}
	svc, err := authSvc.NewService(logger, mongoClient, dataSvcClient)
	if err != nil {
		fmt.Println("Error creating auth service: ", err)
		return
	}
	fmt.Println("[Auth service started]")
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
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
		logger.Log("Listening for signals")
		errChan <- logger.Log(<-signalCh)
	}()
	logger.Log(<-errChan)
}
