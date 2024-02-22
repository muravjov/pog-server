package grpcproxy

import (
	"fmt"
	"net/http"
	"strconv"

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

func checkProxyAuth(r *http.Request, authLst []AuthItem) (string, error) {
	if len(authLst) == 0 {
		return "anonymous", nil
	}

	value, ok := r.Header["Proxy-Authorization"]
	if !ok {
		return "", fmt.Errorf("Proxy-Authorization header required")
	}

	return isAuthenticated(value[0], authLst)
}

func handleTunneling(w http.ResponseWriter, r *http.Request, pcc *ProxyClientContext) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	user := "-"
	logReq := func(code int) {
		logRequest(LogRecord{
			ConnectAddr: r.Host,
			User:        user,
			RemoteAddr:  r.RemoteAddr,
			Code:        strconv.Itoa(code),
		})
	}

	httpErrorAndLog := func(w http.ResponseWriter, errMsg string, code int) {
		httpError(w, errMsg, code)
		logReq(code)
	}

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

		httpErrorAndLog(w, errMsg, code)
	}

	user, err := checkProxyAuth(r, pcc.AuthLst)
	if err != nil {
		w.Header().Set("Proxy-Authenticate", `Basic realm="CLIENT_AUTH_* list"`)
		httpErrorAndLog(w, err.Error(), http.StatusProxyAuthRequired)
		return
	}

	client := pcc.Client

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
		httpErrorAndLog(w, err.Error, int(err.StatusCode))
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
		httpErrorAndLog(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer clientConn.Close()
	logReq(http.StatusOK)

	handleBinaryTunneling(stream, clientConn, cancel)
}

func ProxyHandler(w http.ResponseWriter, r *http.Request, pcc *ProxyClientContext) {
	if r.Method == http.MethodConnect {
		handleTunneling(w, r, pcc)
		return
	}

	if HandleMux(w, r, pcc.MetricsMux) {
		return
	}

	// :TODO:
	//handleHTTP(w, r, client)
	httpError(w, "Not implemented", http.StatusNotImplemented)
}

type ProxyClientContext struct {
	Client  pb.HTTPProxyClient
	AuthLst []AuthItem

	MetricsMux *http.ServeMux
}

func NewProxyClientContext(client pb.HTTPProxyClient) (*ProxyClientContext, error) {
	authLst, err := ParseAuthList(ClientAuthEnvVarPrefix)
	if err != nil {
		return nil, err
	}
	return &ProxyClientContext{
		Client:  client,
		AuthLst: authLst,
	}, nil
}
