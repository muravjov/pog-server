package grpcproxy

import (
	"fmt"
	"net/http"

	pb "git.catbo.net/muravjov/go2023/grpcproxy/proto/v1"
	"git.catbo.net/muravjov/go2023/util"

	"context"
)

func handleTunneling(w http.ResponseWriter, r *http.Request, client pb.HTTPProxyClient) {
	ctx := context.Background()

	bailOut := func(errMsg string, a ...any) {
		http.Error(w, fmt.Sprintf(errMsg, a...), http.StatusInternalServerError)
		return
	}

	stream, err := client.Run(ctx)
	if err != nil {
		bailOut("grpc connection failed: %v", err)
		return
	}
	defer func() {
		if err := stream.CloseSend(); err != nil {
			util.Errorf("stream.CloseSend failed: %v", err)
		}
	}()

	hostPort := r.Host
	packet := &pb.Packet{
		Union: &pb.Packet_ConnectRequest{
			ConnectRequest: &pb.ConnectRequest{
				HostPort: hostPort,
			},
		},
	}
	if err := Send(stream, packet); err != nil {
		bailOut("grpc i/o failure: %v", err)
		return
	}

	pktResp, err := Recv(stream)
	if err != nil {
		bailOut("grpc i/o failure: %v", err)
		return
	}

	resp, err := castFromUnion[*pb.Packet_ConnectResponse](pktResp)
	if err != nil {
		bailOut(err.Error())
		return
	}

	if err := resp.ConnectResponse.Error; err != nil {
		http.Error(w, err.Error, int(err.StatusCode))
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		bailOut("Hijacking not supported")
		return
	}
	// :TRICKY: we need to set status before Hijack() or get an error
	w.WriteHeader(http.StatusOK)

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer clientConn.Close()

	handleBinaryTunneling(stream, clientConn)
}

func ProxyHandler(w http.ResponseWriter, r *http.Request, client pb.HTTPProxyClient) {
	if r.Method == http.MethodConnect {
		handleTunneling(w, r, client)
		return
	}

	// :TODO:
	//handleHTTP(w, r, client)
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}
