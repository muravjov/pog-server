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

	pb "git.catbo.net/muravjov/go2023/grpcproxy/proto/v1"
	"git.catbo.net/muravjov/go2023/grpctest"
	"git.catbo.net/muravjov/go2023/util"
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

func TestProxy(t *testing.T) {
	//t.SkipNow()

	util.SetupSlog(true)

	server := grpc.NewServer()
	// registering should be done before grpcapi.Server.Start() = grpc.Server.Serve()
	RegisterProxySvc(server)

	sc, err := grpctest.StartServerClient(server)
	require.NoError(t, err)
	defer sc.Close()

	//util.Info("grpc server listening on:", sc.Addr.String())

	client := pb.NewHTTPProxyClient(sc.Conn)

	if false {
		for range iter.N(10) {
			// one session per http request being proxied
			ProxySession(client)
		}
	}

	u := "https://ifconfig.me"
	if false {
		u = "https://ifconfig.me1"
	}

	pcc, err := NewProxyClientContext(client)
	require.NoError(t, err)

	if true {
		endpoint := util.Endpoint{
			URL: u,
			Handler: func(w http.ResponseWriter, r *http.Request) {
				//proxy.ProxyHandler(w, r)
				ProxyHandler(w, r, pcc)
			},
		}

		endpoint.Transport = &http.Transport{
			Proxy: func(r *http.Request) (*url.URL, error) {
				u := endpoint.OriginalURL

				var proxy *url.URL
				proxy, err := url.Parse(u)
				require.NoError(t, err)

				// 407
				//proxy.User = url.UserPassword("user", "password")

				// success
				proxy.User = url.UserPassword("user2", "user2")

				return proxy, nil
			},
		}

		util.InvokeEndpoint(&endpoint, false, t)
	}
}
