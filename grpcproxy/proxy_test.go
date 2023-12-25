package grpcproxy

import (
	"testing"

	"github.com/bradfitz/iter"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"git.catbo.net/muravjov/go2023/grpcapi"
	pb "git.catbo.net/muravjov/go2023/grpcproxy/proto/v1"
	"git.catbo.net/muravjov/go2023/grpctest"
)

func TestProxy(t *testing.T) {
	//t.SkipNow()

	server := grpc.NewServer()
	RegisterProxySvc(server)

	s := grpcapi.NewServer(server)

	listener := grpctest.NewLocalListener()

	s.Start(listener)
	defer s.Stop()

	serverAddr := listener.Addr().String()
	//util.Info("grpc server listening on:", serverAddr)

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn, err := grpc.Dial(serverAddr, opts...)
	require.NoError(t, err)
	defer conn.Close()
	client := pb.NewHTTPProxyClient(conn)

	for range iter.N(10) {
		// one session per http request being proxied
		ProxySession(client)
	}
}
