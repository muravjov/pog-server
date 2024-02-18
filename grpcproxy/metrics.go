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

func EnableGRPCServerMetrics(opts []grpc.ServerOption, appRegisterer *prometheus.Registry) (func(), error) {
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
		return nil, err
	}

	if err := view.Register(opencensus.DefaultServerViews...); err != nil {
		util.Errorf("view.Register failed: %v", err)
		return nil, err
	}

	// `Deprecated: in lieu of metricexport.Reader interface.` (for Prometheus exporter only)
	//view.RegisterExporter(exporter)
	//defer view.UnregisterExporter(exporter)
	_ = exporter

	return func() {
		view.Unregister(opencensus.DefaultServerViews...)
	}, nil
}
