//
// Copyright 2019
//
// @author: Denys Nahurnyi
// @email:  dnahurnyi@gmail.com
// ---------------------------------------------------------------------------
package grpcutils

import (
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/grpc-ecosystem/go-grpc-middleware/retry"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type DialConfig struct {
	//Need retry logic for this particular connection
	UseRetry bool
	//Wait for the backoff function. Default is 50ms
	WaitBetween time.Duration
	//Jitter for the backoff function. Default is 0
	JitterFraction float64
	//GRPC codes for which retry should tried. Default is ResourceExhausted and Unavailable
	RetryCodes []codes.Code
	//Per retry timeout. Default is 0, which means disabled
	PerRetryTimeout time.Duration
	//Max retries. Default is 0, which means disabled
	MaxRetries uint
	//GRPC dial options
	DialOptions []grpc.DialOption
}

//This func does grpc dials with the retry dialoption (if needed) + grpc dial options
func GrpcDial(target string, config DialConfig) (conn *grpc.ClientConn, err error) {
	var (
		callOption          []grpc_retry.CallOption
		useDefault          bool            = true
		grpcRetryDialOption grpc.DialOption = nil
	)
	if config.UseRetry == true {
		if config.WaitBetween > 0 {
			useDefault = false
			if config.JitterFraction == 0 {
				callOption = append(callOption, grpc_retry.WithBackoff(grpc_retry.BackoffLinear(config.WaitBetween)))
			} else {
				callOption = append(callOption, grpc_retry.WithBackoff(grpc_retry.BackoffLinearWithJitter(config.WaitBetween, config.JitterFraction)))
			}
		}
		if config.RetryCodes != nil {
			useDefault = false
			callOption = append(callOption, grpc_retry.WithCodes(config.RetryCodes...))
		}
		if config.PerRetryTimeout > 0 {
			useDefault = false
			callOption = append(callOption, grpc_retry.WithPerRetryTimeout(config.PerRetryTimeout))
		}
		if config.MaxRetries > 0 {
			useDefault = false
			callOption = append(callOption, grpc_retry.WithMax(config.MaxRetries))
		}
		if useDefault == true {
			grpcRetryDialOption = grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor())
		} else {
			grpcRetryDialOption = grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(callOption...))
		}
	}
	if grpcRetryDialOption != nil {
		if config.DialOptions != nil {
			config.DialOptions = append(config.DialOptions, grpcRetryDialOption)
		} else {
			config.DialOptions = []grpc.DialOption{grpcRetryDialOption}
		}
		conn, err = grpc.Dial(target, config.DialOptions...)
	} else {
		if config.DialOptions != nil {
			conn, err = grpc.Dial(target, config.DialOptions...)
		} else {
			conn, err = grpc.Dial(target)
		}
	}
	return
}

//This is a wrapper on top of GrpcDial, which does dns lookup as well
func CreateGrpcConn(serviceName, portName string, config DialConfig, logger log.Logger) (conn *grpc.ClientConn, err error) {
	//Discover the service endpoint
	ns := os.Getenv("NAMESPACE")
	var name string
	if ns != "" {
		name = serviceName + "-service." + ns + ".svc.cluster.local"
	} else {
		name = serviceName + "-service.default.svc.cluster.local"
	}
	_, rec, err := net.LookupSRV(portName, "tcp", name)
	if err != nil {
		// msg := "Failed to find service " + serviceName + " port " + portName
		return nil, err
	}

	grpcAddr := rec[0].Target + ":" + strconv.Itoa(int(rec[0].Port))
	conn, err = GrpcDial(grpcAddr, config)
	if err != nil {
		return nil, err
	}
	return conn, err
}

//Creates a newGRPC Server with max recv and send buffer size
func NewServer() *grpc.Server {
	return grpc.NewServer(grpc.MaxRecvMsgSize(math.MaxInt32), grpc.MaxSendMsgSize(math.MaxInt32))
}

//Returns grpc default options for grpc gateway
func OptsGrpcGw() []grpc.DialOption {
	return []grpc.DialOption{grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32))}
}

func forLogs(strRaw interface{}) string {
	str := fmt.Sprintf("%v", strRaw)
	for strings.LastIndex(str, " ") != -1 {
		spaceIn := strings.LastIndex(str, " ")
		if len(str)-1 == spaceIn { // Space is last char in str
			str = str[:spaceIn]
			continue
		}
		if len(str) == spaceIn { // Space is prelast char in str
			str = str[:spaceIn] + strings.ToUpper(string(str[spaceIn+1]))
			continue
		}
		str = str[:spaceIn] + strings.ToUpper(string(str[spaceIn+1])) + str[spaceIn+2:]

	}
	return str
}
