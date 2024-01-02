package main

import (
	"net"
	"os"

	"git.catbo.net/muravjov/go2023/grpcapi"
	"git.catbo.net/muravjov/go2023/grpcproxy"
	"git.catbo.net/muravjov/go2023/util"
	"google.golang.org/grpc"
)

func main() {
	exitCode := 0
	if !Main() {
		exitCode = 1
	}

	os.Exit(exitCode)
}

var Version = "dev"

func Main() bool {
	util.Infof("proxy-over-grpc server, version: %s", Version)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	util.Infof("starting on port %s", port)
	util.Infof("PID: %v", os.Getpid())

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		util.Errorf("net.Listen: %v", err)
		return false
	}

	server := grpc.NewServer()
	grpcproxy.RegisterProxySvc(server)

	return grpcapi.StartAndStop(server, listener, func() {})
}
