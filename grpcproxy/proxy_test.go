package grpcproxy

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"testing"

	"github.com/bradfitz/iter"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"git.catbo.net/muravjov/go2023/grpcapi"
	pb "git.catbo.net/muravjov/go2023/grpcproxy/proto/v1"
	"git.catbo.net/muravjov/go2023/grpctest"
	"git.catbo.net/muravjov/go2023/util"
)

func handleTunneling(w http.ResponseWriter, r *http.Request, client pb.HTTPProxyClient) {
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

	hostPort := r.Host

	// :TODO!!!: set http.Error on Send/Recv error etc

	packet := &pb.Packet{
		Union: &pb.Packet_ConnectRequest{
			ConnectRequest: &pb.ConnectRequest{
				HostPort: hostPort,
			},
		},
	}
	if err := Send(stream, packet); err != nil {
		return
	}

	resp, err := Recv(stream)
	if err != nil {
		return
	}

	log.Printf("Got reponse %s ", resp)

	// :TODO:
	http.Error(w, "Not implemented", http.StatusNotImplemented)
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

func TestProxy(t *testing.T) {
	//t.SkipNow()

	util.SetupSlog(true)

	server := grpc.NewServer()
	RegisterProxySvc(server)

	s := grpcapi.NewServer(server)

	listener := grpctest.NewLocalListener()

	s.Start(listener)
	defer s.Stop()

	serverAddr := listener.Addr().String()
	//util.Info("grpc server listening on:", serverAddr)

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn, err := grpc.Dial(serverAddr, opts...)
	require.NoError(t, err)
	defer conn.Close()
	client := pb.NewHTTPProxyClient(conn)

	if false {
		for range iter.N(10) {
			// one session per http request being proxied
			ProxySession(client)
		}
	}

	if true {
		endpoint := util.Endpoint{
			URL: "https://ifconfig.me",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				//proxy.ProxyHandler(w, r)
				ProxyHandler(w, r, client)
			},
		}

		endpoint.Transport = &http.Transport{
			Proxy: func(r *http.Request) (*url.URL, error) {
				u := endpoint.OriginalURL

				var proxy *url.URL
				proxy, err := url.Parse(u)
				require.NoError(t, err)

				return proxy, nil
			},
		}

		util.InvokeEndpoint(&endpoint, false, t)
	}

}
