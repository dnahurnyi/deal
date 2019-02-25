//
// Copyright 2019
//
// @author: Denys Nahurnyi
// @email:  dnahurnyi@gmail.com
// ---------------------------------------------------------------------------
package utils

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	// "log"
	"net"
	"strings"

	grpcutils "github.com/DenysNahurnyi/deal/common/grpc"
	pb "github.com/DenysNahurnyi/deal/pb/generated/pb"
	"github.com/go-kit/kit/log"
	"github.com/tidwall/gjson"
)

const (
	INVENTORYSERVICE             = "policyreader"
	INVENTORYSERVICEPORTNAME     = "grpc-port"
	GRAPHQUERYSERVICE            = "graphquerymgr"
	GRAPHQUERYSERVICEPORTNAME    = "grpc-port"
	CALENDARSERVICE              = "calendar"
	GRPCMAXRETRIES               = 5
	GRPCPERRETRYTIMEOUT          = 5 * time.Second
	GRPCWAITTIMEOUT              = 5 * time.Second
	INTENTSERVICE                = "intentmgr"
	INTENTSERVICEPORTNAME        = "grpc-port"
	ALERTMGRSERVICE              = "alertmgr"
	ALERTMGRSERVICEPORTNAME      = "grpc-port"
	CHANGEMGRSERVICE             = "changemgr"
	CHANGEMGRSERVICEPORTNAME     = "grpc-port"
	TENANTMGRSERVICE             = "tenantmgr"
	DATA_SERVICE                 = "datasvc"
	WATCHER_SERVICE              = "watchersvc"
	AUTH_SERVICE                 = "authsvc"
	TENANTMGRSERVICEPORTNAME     = "grpc-port"
	DATA_SERVICE_PORT_NAME       = "grpc-port"
	WATCHER_SERVICE_PORT_NAME    = "grpc-port"
	AUTH_SERVICE_PORT_NAME       = "grpc-port"
	GRAPHDBSERVICE               = "graphdbmgr"
	GRAPHDBSERVICEPORTNAME       = "grpc-port"
	SKYNETSERVICE                = "skynet"
	SKYNETSERVICEPORTNAME        = "grpc-port"
	LOGCONTROLLERSERVICE         = "logcontroller"
	LOGCONTROLLERSERVICEPORTNAME = "grpc-port"
	USAGEMANAGERSERVICE          = "usagemanager"
	USAGEMANAGERSERVICEPORTNAME  = "grpc-port"
	ONBOARDINGSERVICE            = "onboarding"
	ONBOARDINGPORTNAME           = "grpc-port"
)

//Checks if a given string exists in the slice
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

//Checks if a given string exists in the slice, ignore case
func StringInSliceIgnoreCase(a string, list []string) bool {
	for _, b := range list {
		if strings.EqualFold(b, a) {
			return true
		}
	}
	return false
}

func StringInJSONSlice(a string, list []gjson.Result) bool {
	for _, b := range list {
		if a == b.String() {
			return true
		}
	}
	return false
}

//Remove duplicate strings in the string slice.
func UniqueStringSlice(StringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range StringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// merge map2 into map1
func MergeMapofStringSlice(map1 map[string][]string, map2 map[string][]string) {
	if map1 == nil {
		map1 = make(map[string][]string)
	}

	for k, v := range map2 {
		if _, ok := map1[k]; !ok {
			map1[k] = v
		} else {
			map1[k] = append(map1[k], v...)
			// remove if any duplicates.
			map1[k] = UniqueStringSlice(map1[k])
		}
	}
}

// Add keys from slice to the given map.
func AddSliceItemsToMap(map1 map[string]bool, list1 []string) {
	if map1 == nil {
		map1 = make(map[string]bool)
	}

	for _, key := range list1 {
		map1[key] = true
	}
}

// //Checks if a given string exists in the slice
// func EnumInSlice(a pb.ResourceTypeEnum, list []pb.ResourceTypeEnum) bool {
// 	for _, b := range list {
// 		if b == a {
// 			return true
// 		}
// 	}
// 	return false
// }

// Check if IP address belongs to a network (CIDR block)
func IsIPAddressInNetwork(network string, ipAddress string) bool {
	_, ParsedNetwork, _ := net.ParseCIDR(network)
	ParsedIPAddress := net.ParseIP(ipAddress)
	if ParsedNetwork.Contains(ParsedIPAddress) {
		return true
	}

	return false
}

// Finds elements in slice1 which aren't in slice2
func SliceDifference(slice1 []string, slice2 []string) []string {
	map1 := map[string]bool{}
	for _, a := range slice2 {
		map1[a] = true
	}

	newslice := []string{}
	for _, a := range slice1 {
		if _, ok := map1[a]; !ok {
			newslice = append(newslice, a)
		}
	}

	return newslice
}

//CreateAuthSvcClient creates a connection to authSvc
func CreateAuthSvcClient(logger log.Logger) (*pb.AuthServiceClient, error) {
	//Discover the authSvc endpoint
	conn, err := grpcutils.CreateGrpcConn(AUTH_SERVICE, AUTH_SERVICE_PORT_NAME, grpcutils.DialConfig{
		UseRetry:    true,
		MaxRetries:  GRPCMAXRETRIES,
		DialOptions: grpcutils.OptsGrpcGw(),
	}, logger)
	if err != nil {
		fmt.Println("grpc Dial failed: ", err)
		return nil, err
	}
	authSvcClient := pb.NewAuthServiceClient(conn)
	return &authSvcClient, nil
}

//CreateDataSvcClient creates a connection to dataSvc
func CreateDataSvcClient(logger log.Logger) (*pb.DataServiceClient, error) {
	//Discover the dataSvc endpoint
	conn, err := grpcutils.CreateGrpcConn(DATA_SERVICE, DATA_SERVICE_PORT_NAME, grpcutils.DialConfig{
		UseRetry:    true,
		MaxRetries:  GRPCMAXRETRIES,
		DialOptions: grpcutils.OptsGrpcGw(),
	}, logger)
	if err != nil {
		fmt.Println("grpc Dial failed: ", err)
		return nil, err
	}
	dataSvcClient := pb.NewDataServiceClient(conn)
	return &dataSvcClient, nil
}

//CreateWatcherSvcClient creates a connection to watcherSvc
func CreateWatcherSvcClient(logger log.Logger) (*pb.WatcherServiceClient, error) {
	//Discover the watcherSvc endpoint
	conn, err := grpcutils.CreateGrpcConn(WATCHER_SERVICE, WATCHER_SERVICE_PORT_NAME, grpcutils.DialConfig{
		UseRetry:    true,
		MaxRetries:  GRPCMAXRETRIES,
		DialOptions: grpcutils.OptsGrpcGw(),
	}, logger)
	if err != nil {
		fmt.Println("grpc Dial failed: ", err)
		return nil, err
	}
	watcherSvcClient := pb.NewWatcherServiceClient(conn)
	return &watcherSvcClient, nil
}

// //Remove duplicate elements in the object list and generate unique alert tags.
// func GenerateUniqueAlertTags(StructSlice []pb.ObjectDetails) ([]*pb.ObjectDetails, []string) {
// 	keys := make(map[string][]string)
// 	var newlist []*pb.ObjectDetails
// 	var tags []string

// 	for _, entry := range StructSlice {
// 		if !StringInSlice(entry.Name, keys[entry.Type]) {
// 			keys[entry.Type] = append(keys[entry.Type], entry.Name)
// 			newlist = append(newlist, &pb.ObjectDetails{
// 				Type:      entry.Type,
// 				Name:      entry.Name,
// 				AccountId: entry.AccountId})
// 		}
// 	}

// 	for tag := range keys {
// 		tags = append(tags, tag)
// 	}

// 	return newlist, tags
// }

// Use for some variables that could contain spaces and just for strings to increase clarity, could be instead of comments
func ForLogs(strRaw interface{}) string {
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

// CheckCommandExecution checks out return code of an external program
func CheckCommandExecution(logger log.Logger, output []byte, err error) {
	if err != nil {
		logger.Log("commandOutput", string(output))
		panic(err)
	}
}

// ReadLine reads long lines by resolving the isPrefix hint from standard ReadLine
func ReadLine(r *bufio.Reader) (string, error) {
	var (
		isPrefix bool  = true
		err      error = nil
		line, ln []byte
	)
	for isPrefix && err == nil {
		line, isPrefix, err = r.ReadLine()
		ln = append(ln, line...)
	}
	return string(ln), err
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func Max(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}

func Min(x, y int64) int64 {
	if x > y {
		return y
	}
	return x
}

func MakeS3Path(keys ...string) string {
	return strings.Join(keys, "/")
}

func GetServiceNameEnv() (string, bool) {
	var ok bool
	var host string
	if host, ok = os.LookupEnv("HOSTNAME"); ok {
		sliceHostParts := strings.Split(host, "-")
		if len(sliceHostParts) > 0 {
			return sliceHostParts[0], true
		}
	}
	return "", false
}
