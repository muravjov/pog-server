package grpcproxy

import (
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
	"git.catbo.net/muravjov/go2023/proxy"
	"git.catbo.net/muravjov/go2023/util"
)

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
				proxy.ProxyHandler(w, r)
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
