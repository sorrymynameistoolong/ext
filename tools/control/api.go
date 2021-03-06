package control

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"

	logService "v2ray.com/core/app/log/command"
	statsService "v2ray.com/core/app/stats/command"
	"v2ray.com/core/common"
)

type ApiCommand struct{}

func (c *ApiCommand) Name() string {
	return "api"
}

func (c *ApiCommand) Description() Description {
	return Description{
		Short: "Call V2Ray API",
		Usage: []string{
			"v2ctl api [--server=127.0.0.1:8080] Service.Method Request",
			"Call an API in an V2Ray process.",
			"The following methods are currently supported:",
			"\tLoggerService.RestartLogger",
			"\tStatsService.GetStats",
		},
	}
}

func (c *ApiCommand) Execute(args []string) error {
	fs := flag.NewFlagSet(c.Name(), flag.ContinueOnError)

	serverAddrPtr := fs.String("server", "127.0.0.1:8080", "Server address")

	if err := fs.Parse(args); err != nil {
		return err
	}

	conn, err := grpc.Dial(*serverAddrPtr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return newError("failed to dial ", *serverAddrPtr).Base(err)
	}
	defer conn.Close()

	unnamedArgs := fs.Args()
	if len(unnamedArgs) < 2 {
		return newError("service name or request not specified.")
	}

	service, method := getServiceMethod(unnamedArgs[0])
	handler, found := serivceHandlerMap[strings.ToLower(service)]
	if !found {
		return newError("unknown service: ", service)
	}

	response, err := handler(conn, method, unnamedArgs[1])
	if err != nil {
		return newError("failed to call service ", unnamedArgs[0]).Base(err)
	}

	fmt.Println(response)
	return nil
}

func getServiceMethod(s string) (string, string) {
	ss := strings.Split(s, ".")
	service := ss[0]
	var method string
	if len(ss) > 1 {
		method = ss[1]
	}
	return service, method
}

type serviceHandler func(conn *grpc.ClientConn, method string, request string) (string, error)

var serivceHandlerMap = map[string]serviceHandler{
	"statsservice":  callStatsService,
	"loggerservice": callLogService,
}

func callLogService(conn *grpc.ClientConn, method string, request string) (string, error) {
	client := logService.NewLoggerServiceClient(conn)

	switch strings.ToLower(method) {
	case "restartlogger":
		r := &logService.RestartLoggerRequest{}
		if err := proto.UnmarshalText(request, r); err != nil {
			return "", err
		}
		resp, err := client.RestartLogger(context.Background(), r)
		if err != nil {
			return "", err
		}
		return proto.MarshalTextString(resp), nil
	default:
		return "", errors.New("Unknown method: " + method)
	}
}

func callStatsService(conn *grpc.ClientConn, method string, request string) (string, error) {
	client := statsService.NewStatsServiceClient(conn)

	switch strings.ToLower(method) {
	case "getstats":
		r := &statsService.GetStatsRequest{}
		if err := proto.UnmarshalText(request, r); err != nil {
			return "", err
		}
		resp, err := client.GetStats(context.Background(), r)
		if err != nil {
			return "", err
		}
		return proto.MarshalTextString(resp), nil
	default:
		return "", errors.New("Unknown method: " + method)
	}
}

func init() {
	common.Must(RegisterCommand(&ApiCommand{}))
}
