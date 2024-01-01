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

	u := "https://ifconfig.me"
	if false {
		u = "https://ifconfig.me1"
	}

	if true {
		endpoint := util.Endpoint{
			URL: u,
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
