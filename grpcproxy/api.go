package grpcproxy

import (
	pb "git.catbo.net/muravjov/go2023/grpcproxy/proto/v1"
	"git.catbo.net/muravjov/go2023/util"
)

type Stream interface {
	Send(*pb.Packet) error
	Recv() (*pb.Packet, error)
}

func Send(stream Stream, packet *pb.Packet) error {
	if err := stream.Send(packet); err != nil {
		util.Errorf("stream.Send(%v) failed: %v", packet, err)
		return err
	}
	return nil
}

func Recv(stream Stream) (*pb.Packet, error) {
	packet, err := stream.Recv()
	if err != nil {
		util.Errorf("stream.Recv(%v) failed: %v", packet, err)
	}
	return packet, err
}
