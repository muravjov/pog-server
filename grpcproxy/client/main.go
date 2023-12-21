package main

import (
	"context"
	"crypto/tls"
	"log"

	pb "git.catbo.net/muravjov/go2023/grpcproxy/proto/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
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

	proxySession(client)
}

func proxySession(client pb.HTTPProxyClient) {
	ctx := context.Background()

	stream, err := client.Run(ctx)
	if err != nil {
		log.Printf("client.Run failed: %v", err)
		return
	}
	defer func() {
		if err := stream.CloseSend(); err != nil {
			log.Printf("stream.CloseSend failed: %v", err)
		}
	}()

	// :TODO!!!:
	hostPort := "ifconfig.me:443"

	packet := &pb.Packet{
		Union: &pb.Packet_ConnectRequest{
			ConnectRequest: &pb.ConnectRequest{
				HostPort: hostPort,
			},
		},
	}
	if err := stream.Send(packet); err != nil {
		log.Printf("client.Run: stream.Send(%v) failed: %v", packet, err)
		return
	}

	resp, err := stream.Recv()
	if err != nil {
		log.Printf("client.Run: stream.Recv() failed: %v", err)
		return
	}

	log.Printf("Got reponse %s ", resp)
}
