package main

import (
	"net"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"

	"git.catbo.net/muravjov/go2023/grpcapi"
	"git.catbo.net/muravjov/go2023/grpcproxy"
	"git.catbo.net/muravjov/go2023/gstacks"
	"git.catbo.net/muravjov/go2023/healthcheck"
	"git.catbo.net/muravjov/go2023/util"
)

func main() {
	exitCode := 0
	if !Main() {
		exitCode = 1
	}

	os.Exit(exitCode)
}

var Version = "dev"

func Main() bool {
	util.Infof("proxy-over-grpc server, version: %s", Version)
	startTimestamp := time.Now()

	appRegisterer := prometheus.NewRegistry()
	util.TryRegisterAppMetrics(appRegisterer)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	util.Infof("starting on port %s", port)
	util.Infof("PID: %v", os.Getpid())

	opts := []grpc.ServerOption{}

	if grpcproxy.IsGRPCBuiltinMetricsEnabled() {
		unregister, err := grpcproxy.EnableGRPCServerMetrics(opts, appRegisterer)
		if err != nil {
			return false
		}
		defer unregister()
	}

	authLst, err := grpcproxy.ParseAuthList(grpcproxy.POGAuthEnvVarPrefix)
	if err != nil {
		return false
	}

	if len(authLst) > 0 {
		ai := &grpcproxy.AuthInterceptor{AuthLst: authLst}
		opts = append(opts, grpc.ChainUnaryInterceptor(ai.ProcessUnary), grpc.ChainStreamInterceptor(ai.ProcessStream))
	}

	server := grpc.NewServer(opts...)
	grpcproxy.RegisterProxySvc(server)
	healthcheck.RegisterHealthcheckSvc(server, "proxy-over-grpc server", startTimestamp, Version)
	gstacks.RegisterGStacksSvc(server)

	serverListen := ":" + port

	var grpcAndHTTPMux bool
	util.BoolEnv(&grpcAndHTTPMux, "GRPC_AND_HTTP_MUX", true)
	if grpcAndHTTPMux {
		httpMux := grpcproxy.NewMetricsMux(appRegisterer)

		mixedHandler := newHTTPandGRPCMux(httpMux, server)
		http2Server := &http2.Server{}
		http1Server := &http.Server{
			Addr:    serverListen,
			Handler: h2c.NewHandler(mixedHandler, http2Server),
		}

		return util.ListenAndServe(http1Server, func() {})
	}

	listener, err := net.Listen("tcp", serverListen)
	if err != nil {
		util.Errorf("net.Listen: %v", err)
		return false
	}

	return grpcapi.StartAndStop(server, listener, func() {})
}
