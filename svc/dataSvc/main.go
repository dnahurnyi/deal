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

	"github.com/DenysNahurnyi/deal/common/grpc"
	dataSvc "github.com/DenysNahurnyi/deal/dataSvc"
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
	grpcPort := ":8010"
	errChan := make(chan error)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	client, err := mongo.Connect(ctx, "mongodb://n826:qwerty12345@ds055732.mlab.com:55732/travel")
	if err != nil {
		fmt.Println("Failed to connect to mongo, err: ", err)
		return
	}

	svc, err := dataSvc.NewService(logger, client)
	if err != nil {
		fmt.Println("Failed to create new data service: ", err)
		return
	}
	go func() {
		listener, err := net.Listen("tcp", grpcPort)
		if err != nil {
			errChan <- err
			return
		}
		handler := dataSvc.NewGRPCServer(svc, logger)
		gRPCServer := grpcutils.NewServer()
		pb.RegisterDataServiceServer(gRPCServer, handler)

		errChan <- gRPCServer.Serve(listener)
	}()

	go func() {
		opts := grpcutils.OptsGrpcGw()
		mux := runtime.NewServeMux()
		err := pb.RegisterDataServiceHandlerFromEndpoint(ctx, mux, grpcPort, opts)
		if err != nil {
			errChan <- err
			return
		}
		errChan <- http.ListenAndServe(":8011", mux)
	}()
	go func() {
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
		logger.Log("Listening for signals")
		errChan <- logger.Log(<-signalCh)
	}()
	logger.Log(<-errChan)
}