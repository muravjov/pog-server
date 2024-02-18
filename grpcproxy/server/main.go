package main

import (
	"net"
	"net/http"
	"os"
	"time"

	promexporter "contrib.go.opencensus.io/exporter/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"go.opencensus.io/stats/view"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats/opencensus"

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

	cfg := struct {
		GRPCAndHTTPMux     bool
		GRPCBuiltinMetrics bool
	}{}
	util.BoolEnv(&cfg.GRPCAndHTTPMux, "GRPC_AND_HTTP_MUX", true)
	util.BoolEnv(&cfg.GRPCBuiltinMetrics, "GRPC_BUILTIN_METRICS", true)

	opts := []grpc.ServerOption{}

	if cfg.GRPCBuiltinMetrics {
		// how it works:
		// - opencensus.ServerOption() enables grpc.StatsHandler to go to census meaures at
		//   google.golang.org/grpc/stats/opencensus@v1.0.0/server_metrics.go
		// - opencensus.DefaultServerViews is a list of metrics (in Prometheus terms)
		// - view.Register(opencensus.DefaultServerViews) registers metrics to start collecting metrics
		//   (as registry.Register() for Prometheus)
		// - promexporter.NewExporter(appRegisterer) registers opencensus metrics in Prometheus
		//   (basically appRegisterer.Register(collector)), see at
		//    /Users/ilya/opt/programming/golang/base-1.21.3/pkg/mod/contrib.go.opencensus.io/exporter/prometheus@v0.4.2/prometheus.go)
		opts = append(opts, opencensus.ServerOption(opencensus.TraceOptions{DisableTrace: true}))

		exporter, err := promexporter.NewExporter(promexporter.Options{Registry: appRegisterer})
		if err != nil {
			util.Errorf("promexporter.NewExporter failed: %v", err)
			return false
		}

		if err := view.Register(opencensus.DefaultServerViews...); err != nil {
			util.Errorf("view.Register failed: %v", err)
			return false
		}
		defer view.Unregister(opencensus.DefaultServerViews...)

		// `Deprecated: in lieu of metricexport.Reader interface.` (for Prometheus exporter only)
		//view.RegisterExporter(exporter)
		//defer view.UnregisterExporter(exporter)
		_ = exporter
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

	if cfg.GRPCAndHTTPMux {
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
