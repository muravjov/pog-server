package grpctest

import (
	"fmt"
	"net"

	"git.catbo.net/muravjov/go2023/grpcapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// stolen from net/http/httptest
func NewLocalListener() net.Listener {
	if serveFlag != "" {
		l, err := net.Listen("tcp", serveFlag)
		if err != nil {
			panic(fmt.Sprintf("httptest: failed to listen on %v: %v", serveFlag, err))
		}
		return l
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if l, err = net.Listen("tcp6", "[::1]:0"); err != nil {
			panic(fmt.Sprintf("httptest: failed to listen on a port: %v", err))
		}
	}
	return l
}

var serveFlag string

type ServerClient struct {
	s *grpcapi.Server

	Conn *grpc.ClientConn
	Addr net.Addr
}

func (s *ServerClient) Close() {
	s.Conn.Close()
	s.s.Stop()
}

func StartServerClient(server *grpc.Server) (*ServerClient, error) {
	s := grpcapi.NewServer(server)

	listener := NewLocalListener()

	s.Start(listener)

	serverAddr := listener.Addr().String()

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		s.Stop()
		return nil, err
	}

	return &ServerClient{
		s:    s,
		Conn: conn,
		Addr: listener.Addr(),
	}, nil
}
