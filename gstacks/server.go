package gstacks

import (
	"context"
	"time"

	pb "git.catbo.net/muravjov/go2023/gstacks/proto/v1"
	"google.golang.org/grpc"
)

func RegisterGStacksSvc(server *grpc.Server) {
	svc := &gstacksServer{}
	pb.RegisterGoroutineStacksServer(server, svc)
}

type gstacksServer struct {
	appName        string
	startTimestamp time.Time
	version        string

	pb.UnimplementedGoroutineStacksServer
}

func (s *gstacksServer) Invoke(context.Context, *pb.Request) (*pb.Response, error) {
	return &pb.Response{Data: goroutineStacks()}, nil
}
