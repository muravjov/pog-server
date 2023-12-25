package grpcapi

import (
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"git.catbo.net/muravjov/go2023/util"
	"google.golang.org/grpc"
)

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
