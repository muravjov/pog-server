package main

import (
	"log"
	"net"
	"os"

	"git.catbo.net/muravjov/go2023/grpcapi"
	"git.catbo.net/muravjov/go2023/grpcproxy"
	"google.golang.org/grpc"
)

func main() {
	exitCode := 0
	if !Main() {
		exitCode = 1
	}

	os.Exit(exitCode)
}

func Main() bool {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("proxy-via-grpc server: starting on port %s", port)
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("net.Listen: %v", err)
	}

	server := grpc.NewServer()
	grpcproxy.RegisterProxySvc(server)

	return grpcapi.StartAndStop(server, listener, func() {})
}
