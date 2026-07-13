package tracking

import (
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"go.uber.org/zap"
)

type PrometheusOpts struct {
	Apdex bool
}

func Prometheus(opts PrometheusOpts) {
	// Create a new registry.
	reg := prometheus.NewRegistry()

	// Stormkit related exports
	if opts.Apdex {
		reg.MustRegister(RTHistogramProdEndpoints)
	}

	// Add Go module build info.
	reg.MustRegister(collectors.NewBuildInfoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	reg.MustRegister(collectors.NewGoCollector(
		collectors.WithGoCollectorRuntimeMetrics(collectors.GoRuntimeMetricsRule{
			Matcher: regexp.MustCompile("/.*"),
		}),
	))

	// Start HTTP server for Prometheus to scrape metrics
	http.Handle("/metrics", promhttp.HandlerFor(
		reg,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
		},
	))

	go func() {
		conf := config.Get()

		for i := 0; i < 10; i++ {
			port := utils.StringToInt(conf.Tracking.PrometheusPort)

			if !utils.IsPortInUse(port) {
				break
			}

			slog.Debug(slog.LogOpts{
				Msg:   "prometheus port is already in use, picking a new one",
				Level: slog.DL1,
				Payload: []zap.Field{
					zap.String("port", conf.Tracking.PrometheusPort),
				},
			})

			conf.Tracking.PrometheusPort = utils.Int64ToString(int64(port + 1))
		}

		slog.Infof("prometheus metrics available at /metrics, port: %s", conf.Tracking.PrometheusPort)
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", conf.Tracking.PrometheusPort), nil))
	}()
}
