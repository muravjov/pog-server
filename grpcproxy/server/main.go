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
	packet, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, "stream.Recv failed: %v", err)
	}

	req, ok := packet.Union.(*pb.Packet_ConnectRequest)
	if !ok {
		return status.Errorf(codes.FailedPrecondition, "ConnectRequest packet expected but got: %v", packet.Union)
	}

	// :TODO!!!:
	log.Printf("CONNECT %v", req.ConnectRequest.HostPort)

	packet = &pb.Packet{
		Union: &pb.Packet_ConnectResponse{
			ConnectResponse: &pb.ConnectResponse{},
		},
	}

	// :REFACTOR:
	if err := stream.Send(packet); err != nil {
		log.Printf("stream.Send(%v) failed: %v", packet, err)
		return err
	}

	// :TODO!!!:
	return status.Errorf(codes.Unimplemented, "not implemented")
}
