package grpcproxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"sync"

	pb "git.catbo.net/muravjov/go2023/grpcproxy/proto/v1"
	"git.catbo.net/muravjov/go2023/util"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Stream interface {
	Send(*pb.Packet) error
	Recv() (*pb.Packet, error)
}

func isEndError(err error) bool {
	// our cancel() errors can be converted to gRPC' Canceled errors
	// - on client: when we call cancel() on a context used for a client stream =>
	//   Send() will err because that client stream is closed
	// - on server: when we got Canceled from client and so Recv() returns Canceled
	if status.Code(err) == codes.Canceled {
		return true
	}

	// - Send(): we have closed the (client) stream and still trying to send
	// - Recv(): the server side closed the stream
	if err == io.EOF {
		return true
	}

	return false
}

func Send(stream Stream, packet *pb.Packet) error {
	if err := stream.Send(packet); err != nil {
		func() {
			if isEndError(err) {
				return
			}

			util.Errorf("stream.Send(%v) failed: %v", packet, err)
		}()
		return err
	}
	return nil
}

func Recv(stream Stream) (*pb.Packet, error) {
	packet, err := stream.Recv()
	if err != nil {
		func() {
			if isEndError(err) {
				return
			}

			util.Errorf("stream.Recv() failed: %v, %v", packet, err)
		}()
	}
	return packet, err
}

func castFromUnion[T any](packet *pb.Packet) (T, error) {
	t, ok := packet.Union.(T)
	if !ok {
		err := fmt.Errorf("got wrong packet type: %+v", packet.Union)
		util.Error(err)
		return t, err
	}

	return t, nil
}

func NewStreamWriter(s Stream) io.Writer {
	return &streamWriter{s}
}

type streamWriter struct {
	s Stream
}

func (t *streamWriter) Write(p []byte) (int, error) {
	// we avoid sending empty payloads to
	// simplify reading
	if len(p) == 0 {
		return 0, nil
	}

	packet := &pb.Packet{
		Union: &pb.Packet_Payload{
			Payload: p,
		},
	}

	if err := Send(t.s, packet); err != nil {
		return 0, err
	}

	return len(p), nil
}

func NewStreamReader(s Stream) io.Reader {
	return &streamReader{s, &bytes.Buffer{}}
}

type streamReader struct {
	s   Stream
	buf *bytes.Buffer
}

func (t *streamReader) Read(p []byte) (int, error) {
	n, err := t.buf.Read(p)
	if err == nil {
		// we are done using buffer only
		// (maybe partionally but it is ok)
		return n, nil
	}

	// buffer is empty
	packet, err := Recv(t.s)
	if err != nil {
		return 0, err
	}

	resp, err := castFromUnion[*pb.Packet_Payload](packet)
	if err != nil {
		return 0, err
	}

	// :TRICKY:
	// - bytes.Buffer never errs, but can panic
	// - we expect len(resp.Payload) > 0
	if len(resp.Payload) == 0 {
		util.Error("got empty payload packet")
	}

	t.buf.Write(resp.Payload)

	return t.buf.Read(p)
}

var TunnelingConnections = util.NewGaugeVecMetric(
	"tunnelling_connections_total",
	"Number of connections tunneling through the proxy.",
	[]string{},
).With(prometheus.Labels{})

func handleBinaryTunneling(stream Stream, conn net.Conn, streamCancel context.CancelFunc) {
	TunnelingConnections.Inc()
	defer TunnelingConnections.Dec()

	var wg sync.WaitGroup

	// when conn-side closes we close writer, and also we need to finish the transfer() goroutine
	// with the reader => we use `cancel` func to close the stream (client) or initiate close action
	// (server: get out of grpc operation loop)
	transfer(NewStreamWriter(stream), conn, &wg, streamCancel)
	transfer(conn, NewStreamReader(stream), &wg, func() {
		conn.Close()
	})

	wg.Wait()
}

func transfer(destination io.Writer, source io.Reader, wg *sync.WaitGroup, cancel context.CancelFunc) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if cancel != nil {
				cancel()
			}
		}()
		io.Copy(destination, source)
	}()
}
