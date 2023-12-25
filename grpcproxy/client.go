package grpcproxy

import (
	pb "git.catbo.net/muravjov/go2023/grpcproxy/proto/v1"

	"context"
	"log"
)

func ProxySession(client pb.HTTPProxyClient) {
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
