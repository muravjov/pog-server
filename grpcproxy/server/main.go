package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	pb "git.catbo.net/muravjov/go2023/grpcproxy/proto/v1"
	"git.catbo.net/muravjov/go2023/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {
	exitCode := 0
	if !Main() {
		exitCode = 1
	}

	os.Exit(exitCode)
}

func Main() bool {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("proxy-via-grpc server: starting on port %s", port)
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("net.Listen: %v", err)
	}

	svc := new(httpProxyServer)
	server := grpc.NewServer()
	pb.RegisterHTTPProxyServer(server, svc)

	return StartAndStop(server, listener, func() {})
}

type Server struct {
	server *grpc.Server
	wg     sync.WaitGroup

	serverOk bool
}

func NewServer(server *grpc.Server) *Server {
	return &Server{
		server:   server,
		serverOk: true,
	}
}

func (s *Server) Start(lis net.Listener) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.server.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			util.Error(err)
			s.serverOk = false
		}
	}()
}

func (s *Server) Stop() bool {
	s.server.GracefulStop()

	s.wg.Wait()
	return s.serverOk
}

func StartAndStop(server *grpc.Server, lis net.Listener, beforeShutdown func()) bool {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	s := NewServer(server)
	s.Start(lis)

	util.Info("waiting for termination signal...")
	sig := <-sigChan
	util.Infof("signal received: %s", sig.String())

	beforeShutdown()

	return s.Stop()
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
