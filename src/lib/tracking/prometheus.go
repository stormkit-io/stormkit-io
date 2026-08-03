package tracking

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const (
	metricsReadHeaderTimeout = 5 * time.Second
	maxPortProbeAttempts     = 10
)

// PrometheusOpts configures the metrics endpoint.
type PrometheusOpts struct {
	// Apdex registers the HTTP response time histogram. Only the hosting
	// service serves customer traffic, so only it enables this.
	Apdex bool

	// Port overrides the configured metrics port. When set, the listener binds
	// it strictly and never probes for a free one.
	Port string
}

// Prometheus starts the metrics endpoint on its own listener and returns the
// server so callers can shut it down.
//
// It never terminates the process. A metrics port that cannot be bound must not
// take down a service that is busy serving traffic.
func Prometheus(opts PrometheusOpts) *http.Server {
	listener := &metricsListener{port: opts.Port}

	if listener.port == "" {
		conf := config.Get()
		listener.port = conf.Tracking.PrometheusPort

		// Only a defaulted port may move. An operator who names a port gets
		// that port or a loud failure, never a silently different one that
		// leaves their scrape config pointing at nothing.
		listener.probe = !conf.Tracking.PrometheusPortExplicit
	}

	ln, err := listener.listen()

	if err != nil {
		slog.Errorf("prometheus metrics disabled, could not listen on port %s: %v", listener.port, err)
		return nil
	}

	srv := &http.Server{
		Handler:           metricsMux(newRegistry(opts)),
		ReadHeaderTimeout: metricsReadHeaderTimeout,
	}

	slog.Infof("prometheus metrics available at /metrics, port: %s", listenerPort(ln))

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Errorf("prometheus metrics server stopped: %v", err)
		}
	}()

	return srv
}

// newRegistry builds a dedicated registry. Collectors that are not registered
// here are never scraped, so every Stormkit metric must be added below.
func newRegistry(opts PrometheusOpts) *prometheus.Registry {
	reg := prometheus.NewRegistry()

	if opts.Apdex {
		reg.MustRegister(RTHistogramProdEndpoints)
	}

	reg.MustRegister(collectors.NewBuildInfoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	reg.MustRegister(collectors.NewGoCollector(
		collectors.WithGoCollectorRuntimeMetrics(collectors.GoRuntimeMetricsRule{
			Matcher: regexp.MustCompile("/.*"),
		}),
	))

	return reg
}

// metricsMux serves the registry on a dedicated mux rather than the default one,
// so /metrics cannot leak onto any other listener that happens to serve
// http.DefaultServeMux.
func metricsMux(reg *prometheus.Registry) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.HandlerFor(
		reg,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
		},
	))

	return mux
}

// metricsListener binds the metrics port.
//
// Probing exists for local development, where goreman runs hosting and
// workerserver as processes on the same host and they would otherwise fight
// over the same port. In Docker each service has its own network namespace, so
// nothing collides and the port stays where the scrape config expects it.
type metricsListener struct {
	port  string
	probe bool
}

func (m *metricsListener) listen() (net.Listener, error) {
	port := utils.StringToInt(m.port)
	attempts := 1

	if m.probe {
		attempts = maxPortProbeAttempts
	}

	var err error

	for i := range attempts {
		var ln net.Listener

		if ln, err = net.Listen("tcp", fmt.Sprintf(":%d", port+i)); err == nil {
			return ln, nil
		}

		if m.probe {
			slog.Infof("prometheus port %d is in use, trying the next one", port+i)
		}
	}

	return nil, err
}

func listenerPort(ln net.Listener) string {
	if addr, ok := ln.Addr().(*net.TCPAddr); ok {
		return utils.Int64ToString(int64(addr.Port))
	}

	return ln.Addr().String()
}
