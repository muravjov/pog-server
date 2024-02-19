package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"git.catbo.net/muravjov/go2023/grpcproxy"
	pb "git.catbo.net/muravjov/go2023/grpcproxy/proto/v1"
	"git.catbo.net/muravjov/go2023/util"
)

func main() {
	exitCode := 0
	if !Main() {
		exitCode = 1
	}

	os.Exit(exitCode)
}

func Main() bool {
	appRegisterer := prometheus.NewRegistry()
	util.TryRegisterAppMetrics(appRegisterer)

	cfg := MakeConfig()

	var opts []grpc.DialOption

	if grpcproxy.IsGRPCBuiltinMetricsEnabled() {
		opts = append(opts, grpcproxy.ClientStatsOption())

		unregister, err := grpcproxy.EnableGRPCClientMetrics(appRegisterer)
		if err != nil {
			return false
		}
		defer unregister()
	}

	if cfg.ServerAddr == "" {
		util.Error("-server is empty")
		return false
	}
	if cfg.ServerHost != "" {
		opts = append(opts, grpc.WithAuthority(cfg.ServerHost))
	}
	if cfg.Insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		cred := credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: cfg.SkipVerify,
		})
		opts = append(opts, grpc.WithTransportCredentials(cred))
	}

	if cfg.ClientPOGAuth != "" {
		starredCreds := cfg.ClientPOGAuth
		i := strings.Index(starredCreds, ":")
		if i > 0 {
			starredCreds = fmt.Sprintf("%s:***", starredCreds[:i])
		}
		util.Infof("using client-server auth %s", starredCreds)

		opts = append(opts, grpc.WithPerRPCCredentials(grpcproxy.BasicAuthCredentials{Auth: cfg.ClientPOGAuth}))
	}

	conn, err := grpc.Dial(cfg.ServerAddr, opts...)
	if err != nil {
		util.Errorf("failed to dial server %s: %v", cfg.ServerAddr, err)
		return false
	}
	defer conn.Close()
	client := pb.NewHTTPProxyClient(conn)
	pcc, err := grpcproxy.NewProxyClientContext(client)
	if err != nil {
		return false
	}

	pcc.MetricsMux = grpcproxy.NewMetricsMux(appRegisterer)

	server := &http.Server{
		Addr: cfg.ClientListen,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			grpcproxy.ProxyHandler(w, r, pcc)
		}),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	util.Infof("proxy-over-grpc client listening address %s", server.Addr)
	util.Infof("PID: %v", os.Getpid())

	return util.ListenAndServe(server, func() {})
}
