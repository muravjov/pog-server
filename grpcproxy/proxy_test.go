package grpcproxy

import (
	"testing"

	"git.catbo.net/muravjov/go2023/grpcapi"
	"git.catbo.net/muravjov/go2023/grpctest"
	"google.golang.org/grpc"
)

func TestProxy(t *testing.T) {
	//t.SkipNow()

	server := grpc.NewServer()
	RegisterProxySvc(server)

	s := grpcapi.NewServer(server)
	s.Start(grpctest.NewLocalListener())

	defer s.Stop()
}
