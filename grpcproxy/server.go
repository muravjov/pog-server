package grpcproxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	pb "git.catbo.net/muravjov/go2023/grpcproxy/proto/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
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
	user := "anonymous"
	streamCtx := stream.(interface {
		Context() context.Context
	}).Context()
	if ca, ok := streamCtx.Value(connectionAuthKey{}).(ConnectionAuthCtx); ok {
		user = ca.User
	}

	connectAddr := "-"
	connectProto := "HTTPS"

	logRequest := func(code codes.Code) {
		remoteAddr := ""
		if p, ok := peer.FromContext(streamCtx); ok {
			remoteAddr = p.Addr.String()
		}

		fmt.Printf("pog: %s %s %s %v [%v] %v\n", connectAddr, user, connectProto, remoteAddr, time.Now().Format(time.RFC3339), code)
	}

	bailOut := func(err error) {
		*statusErr = err
		logRequest(status.Code(err))
	}

	packet, err := Recv(stream)
	if err != nil {
		bailOut(err)
		return
	}

	req, err := castFromUnion[*pb.Packet_ConnectRequest](packet)
	if err != nil {
		bailOut(status.Error(codes.FailedPrecondition, err.Error()))
		return
	}
	connectAddr = req.ConnectRequest.HostPort

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
		bailOut(err)
		return
	}
	defer destConn.Close()

	if err := sendConnectResponse(nil); err != nil {
		bailOut(err)
		return
	}
	logRequest(codes.OK)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		handleBinaryTunneling(stream, destConn, cancel)
	}()

	<-ctx.Done()
}
