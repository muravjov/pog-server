package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
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

	pcc.MetricsMux = (func() *http.ServeMux {
		var muxServerMetrics bool
		util.BoolEnv(&muxServerMetrics, "MUX_SERVER_METRICS", false)
		if !muxServerMetrics {
			return grpcproxy.NewMetricsMux(appRegisterer)
		}

		httpMux := http.NewServeMux()
		// :TRICKY:
		// - we just append server' /metrics to client' /metrics, and so
		//   we have to avoid gzipping (the client body and then not gzipped response)
		// - we have to "EnableOpenMetrics: false", otherwise it adds
		//   "# EOF\n" between metrics bodies
		//
		// use the flag MUX_SERVER_METRICS=true for a reason
		handler := promhttp.HandlerFor(
			appRegisterer,
			promhttp.HandlerOpts{
				EnableOpenMetrics:  false,
				DisableCompression: true,
			},
		).ServeHTTP

		httpMux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
			clientServerMetrics(w, r, metricsCtx{handler, cfg})
		})

		return httpMux
	})()

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

var metricsMuxErrCnt = util.MakeCounterVecFunc(
	"server_client_metrics_mux_errors_total",
	"Number of errors while getting pog server's /metrics",
)

type metricsCtx struct {
	PromHandler http.HandlerFunc
	Cfg         Config
}

func clientServerMetrics(w http.ResponseWriter, r *http.Request, metricsCtx metricsCtx) {
	metricsCtx.PromHandler(w, r)

	metric := dto.Metric{}
	grpcproxy.TunnelingConnections.Write(&metric)
	if metric.GetGauge().GetValue() == 0 {
		return
	}

	cfg := metricsCtx.Cfg

	// ok, we are working => pog server is up anyway
	schema := "https"
	if metricsCtx.Cfg.Insecure {
		schema = "http"
	}

	u := fmt.Sprintf("%s://%s/metrics", schema, cfg.ServerAddr)

	resp, err := http.Get(u)
	if err != nil {
		metricsMuxErrCnt("GET", 1)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		metricsMuxErrCnt("not_2XX", 1)
		return
	}

	if _, err := io.Copy(w, resp.Body); err != nil {
		metricsMuxErrCnt("copy", 1)
		return
	}

	metricsMuxErrCnt("ok", 1)
}
