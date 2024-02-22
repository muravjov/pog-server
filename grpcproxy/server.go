package grpcproxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
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

type LogRecord struct {
	ConnectAddr string
	User        string
	RemoteAddr  string
	Code        string
}

var disableAccessLogging = os.Getenv("DISABLE_ACCESS_LOGGING") != ""

func logRequest(rec LogRecord) {
	if disableAccessLogging {
		return
	}

	connectProto := "HTTPS"
	fmt.Printf("pog: %s %s %s %v [%v] %v\n", rec.ConnectAddr, rec.User, connectProto, rec.RemoteAddr, time.Now().Format(time.RFC3339), rec.Code)
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

	logReq := func(code codes.Code) {
		remoteAddr := "-"
		if p, ok := peer.FromContext(streamCtx); ok {
			remoteAddr = p.Addr.String()
		}

		logRequest(LogRecord{
			ConnectAddr: connectAddr,
			User:        user,
			RemoteAddr:  remoteAddr,
			Code:        code.String(),
		})
	}

	bailOut := func(err error) {
		*statusErr = err
		logReq(status.Code(err))
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
	logReq(codes.OK)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		handleBinaryTunneling(stream, destConn, cancel)
	}()

	<-ctx.Done()
}
