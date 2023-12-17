package main

import (
	"log"
	"net"
	"os"

	pb "git.catbo.net/muravjov/go2023/grpcproxy/proto/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("timeserver: starting on port %s", port)
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("net.Listen: %v", err)
	}

	svc := new(httpProxyServer)
	server := grpc.NewServer()
	pb.RegisterHTTPProxyServer(server, svc)
	if err = server.Serve(listener); err != nil {
		log.Fatal(err)
	}
}

type httpProxyServer struct {
	pb.UnimplementedHTTPProxyServer
}

func (s *httpProxyServer) Run(stream pb.HTTPProxy_RunServer) error {
	// :TODO!!!:
	return status.Errorf(codes.Unimplemented, "method Run not implemented")
}
