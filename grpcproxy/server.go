package grpcproxy

import (
	"log"

	pb "git.catbo.net/muravjov/go2023/grpcproxy/proto/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RegisterProxySvc(server *grpc.Server) {
	svc := new(httpProxyServer)
	pb.RegisterHTTPProxyServer(server, svc)
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
