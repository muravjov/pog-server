package grpcproxy

import (
	"fmt"
	"net/http"

	pb "git.catbo.net/muravjov/go2023/grpcproxy/proto/v1"
	"git.catbo.net/muravjov/go2023/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"context"
)

func httpError(w http.ResponseWriter, errMsg string, code int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// clients mostly hide errd response bodies from user, so let her know with a header
	w.Header().Set("X-Proxy-Over-GRPC-Error", errMsg)

	w.WriteHeader(code)
	fmt.Fprintln(w, errMsg)
}

func handleTunneling(w http.ResponseWriter, r *http.Request, client pb.HTTPProxyClient) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bailOut := func(errMsg string, a ...any) {
		code := http.StatusInternalServerError
		for _, item := range a {
			err, ok := item.(error)
			if !ok {
				continue
			}

			if status.Code(err) == codes.Unavailable {
				code = http.StatusServiceUnavailable
			}
			if status.Code(err) == codes.Unauthenticated {
				code = http.StatusServiceUnavailable
			}
		}

		errMsg = fmt.Sprintf(errMsg, a...)

		httpError(w, errMsg, code)
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
		httpError(w, err.Error, int(err.StatusCode))
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
		httpError(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer clientConn.Close()

	handleBinaryTunneling(stream, clientConn, cancel)
}

func ProxyHandler(w http.ResponseWriter, r *http.Request, client pb.HTTPProxyClient) {
	if r.Method == http.MethodConnect {
		handleTunneling(w, r, client)
		return
	}

	// :TODO:
	//handleHTTP(w, r, client)
	httpError(w, "Not implemented", http.StatusNotImplemented)
}
