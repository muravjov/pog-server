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
	var statusErr error
	doRun(stream, &statusErr)

	return statusErr
}

func doRun(stream Stream, statusErr *error) {
	packet, err := Recv(stream)
	if err != nil {
		return
	}

	req, ok := packet.Union.(*pb.Packet_ConnectRequest)
	if !ok {
		*statusErr = status.Errorf(codes.FailedPrecondition, "ConnectRequest packet expected but got: %v", packet.Union)
		return
	}

	// :TODO!!!:
	log.Printf("CONNECT %v", req.ConnectRequest.HostPort)

	packet = &pb.Packet{
		Union: &pb.Packet_ConnectResponse{
			ConnectResponse: &pb.ConnectResponse{},
		},
	}
	if err := Send(stream, packet); err != nil {
		return
	}

	// :TODO!!!:
	*statusErr = status.Errorf(codes.Unimplemented, "not implemented")
	//*statusErr = nil
}
