package grpcproxy

import (
	"net/http"

	promexporter "contrib.go.opencensus.io/exporter/prometheus"
	"git.catbo.net/muravjov/go2023/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opencensus.io/stats/view"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats/opencensus"
)

func NewMetricsMux(appRegisterer *prometheus.Registry) *http.ServeMux {
	httpMux := http.NewServeMux()

	httpMux.HandleFunc("/metrics", promhttp.HandlerFor(
		appRegisterer,
		promhttp.HandlerOpts{EnableOpenMetrics: true},
	).ServeHTTP)

	return httpMux
}

func HandleMux(w http.ResponseWriter, r *http.Request, mux *http.ServeMux) bool {
	if mux == nil {
		return false
	}

	h, p := mux.Handler(r)
	if p == "" {
		return false
	}

	h.ServeHTTP(w, r)
	return true
}

// how it works:
// - opencensus.ServerOption() enables grpc.StatsHandler to go to census meaures at
//   google.golang.org/grpc/stats/opencensus@v1.0.0/server_metrics.go
// - opencensus.DefaultServerViews is a list of metrics (in Prometheus terms)
// - view.Register(opencensus.DefaultServerViews) registers metrics to start collecting metrics
//   (as registry.Register() for Prometheus)
// - promexporter.NewExporter(appRegisterer) registers opencensus metrics in Prometheus
//   (basically appRegisterer.Register(collector)), see at
//    contrib.go.opencensus.io/exporter/prometheus@v0.4.2/prometheus.go)

func EnableGRPCServerMetrics(opts []grpc.ServerOption, appRegisterer *prometheus.Registry) (func(), error) {
	opts = append(opts, opencensus.ServerOption(opencensus.TraceOptions{DisableTrace: true}))

	return enableMetrics(opencensus.DefaultServerViews, appRegisterer)
}

func enableMetrics(metricsLst []*view.View, appRegisterer *prometheus.Registry) (func(), error) {
	exporter, err := promexporter.NewExporter(promexporter.Options{Registry: appRegisterer})
	if err != nil {
		util.Errorf("promexporter.NewExporter failed: %v", err)
		return nil, err
	}

	if err := view.Register(metricsLst...); err != nil {
		util.Errorf("view.Register failed: %v", err)
		return nil, err
	}

	// `Deprecated: in lieu of metricexport.Reader interface.` (for Prometheus exporter only)
	//view.RegisterExporter(exporter)
	//defer view.UnregisterExporter(exporter)
	_ = exporter

	return func() {
		view.Unregister(metricsLst...)
	}, nil
}

func IsGRPCBuiltinMetricsEnabled() bool {
	var b bool
	util.BoolEnv(&b, "GRPC_BUILTIN_METRICS", true)
	return b
}

func EnableGRPCClientMetrics(opts []grpc.DialOption, appRegisterer *prometheus.Registry) (func(), error) {
	opts = append(opts, opencensus.DialOption(opencensus.TraceOptions{DisableTrace: true}))

	return enableMetrics(opencensus.DefaultClientViews, appRegisterer)
}
