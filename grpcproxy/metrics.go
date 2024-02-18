package grpcproxy

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
