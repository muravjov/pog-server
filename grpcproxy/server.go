package grpcproxy

import (
	"net"
	"net/http"
	"time"

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

	req, err := castFromUnion[*pb.Packet_ConnectRequest](packet)
	if err != nil {
		*statusErr = status.Error(codes.FailedPrecondition, err.Error())
		return
	}

	sendConnectResponse := func(httpErr *pb.HTTPError) error {
		packet = &pb.Packet{
			Union: &pb.Packet_ConnectResponse{
				ConnectResponse: &pb.ConnectResponse{
					Error: httpErr,
				},
			},
		}
		if err := Send(stream, packet); err != nil {
			return err
		}

		return err
	}

	destConn, err := net.DialTimeout("tcp", req.ConnectRequest.HostPort, 10*time.Second)
	if err != nil {
		sendConnectResponse(&pb.HTTPError{
			StatusCode: http.StatusServiceUnavailable,
			Error:      err.Error(),
		})
		return
	}
	defer destConn.Close()

	if err := sendConnectResponse(nil); err != nil {
		return
	}

	handleBinaryTunneling(stream, destConn)
}
