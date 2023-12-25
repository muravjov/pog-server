package main

import (
	"crypto/tls"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"git.catbo.net/muravjov/go2023/grpcproxy"
	pb "git.catbo.net/muravjov/go2023/grpcproxy/proto/v1"
)

func main() {
	cfg := MakeConfig()

	var opts []grpc.DialOption
	if cfg.ServerAddr == "" {
		log.Fatal("-server is empty")
	}
	if cfg.ServerHost != "" {
		opts = append(opts, grpc.WithAuthority(cfg.ServerHost))
	}
	if cfg.Insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		cred := credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: cfg.SkipVerify,
		})
		opts = append(opts, grpc.WithTransportCredentials(cred))
	}

	conn, err := grpc.Dial(cfg.ServerAddr, opts...)
	if err != nil {
		log.Fatalf("failed to dial server %s: %v", cfg.ServerAddr, err)
	}
	defer conn.Close()
	client := pb.NewHTTPProxyClient(conn)

	grpcproxy.ProxySession(client)
}
