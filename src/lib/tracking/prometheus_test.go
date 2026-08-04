package tracking

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stretchr/testify/suite"
)

type PrometheusSuite struct {
	suite.Suite
}

func (s *PrometheusSuite) gathered(opts PrometheusOpts) string {
	server := httptest.NewServer(metricsMux(newRegistry(opts)))
	defer server.Close()

	res, err := http.Get(server.URL + "/metrics")
	s.Require().NoError(err)

	defer res.Body.Close()
	s.Equal(http.StatusOK, res.StatusCode)

	body, err := io.ReadAll(res.Body)
	s.Require().NoError(err)

	return string(body)
}

// freePort binds and releases a port so we know it was available.
func (s *PrometheusSuite) freePort() int {
	ln, err := net.Listen("tcp", ":0")
	s.Require().NoError(err)

	port := ln.Addr().(*net.TCPAddr).Port
	s.Require().NoError(ln.Close())

	return port
}

func (s *PrometheusSuite) Test_ApdexRegisteredOnlyWhenEnabled() {
	// A histogram vec with no observations exports no series at all, so the
	// metric has to be touched before it can be seen in the output.
	RTHistogramProdEndpoints.WithLabelValues(http.MethodGet, "200").Observe(1)

	s.Contains(s.gathered(PrometheusOpts{Apdex: true}), "stormkit_lb_response_time_ms")
	s.NotContains(s.gathered(PrometheusOpts{Apdex: false}), "stormkit_lb_response_time_ms")
}

// Registration is easy to get wrong silently: a collector left out of
// newRegistry is simply never scraped, and the dashboards show "No data".
func (s *PrometheusSuite) Test_DatabasePoolRegisteredOnEveryProfile() {
	db, err := sql.Open("trackingfake", "")
	s.Require().NoError(err)

	defer db.Close()

	previous := database.CurrentConnection()
	database.SetConnection(db)

	defer database.SetConnection(previous)

	for _, opts := range []PrometheusOpts{{Apdex: true}, {Apdex: false}} {
		s.Contains(s.gathered(opts), "stormkit_db_connections")
	}
}

func (s *PrometheusSuite) Test_RuntimeCollectorsAlwaysRegistered() {
	body := s.gathered(PrometheusOpts{})

	s.Contains(body, "go_goroutines")
	s.Contains(body, "process_resident_memory_bytes")
}

func (s *PrometheusSuite) Test_MetricsEndpointServesRegistry() {
	server := httptest.NewServer(metricsMux(newRegistry(PrometheusOpts{Apdex: true})))
	defer server.Close()

	res, err := http.Get(server.URL + "/metrics")
	s.Require().NoError(err)

	defer res.Body.Close()
	s.Equal(http.StatusOK, res.StatusCode)
}

// The metrics handler must not land on the global mux, or it would leak onto
// any other listener that serves http.DefaultServeMux.
func (s *PrometheusSuite) Test_MetricsNotOnDefaultServeMux() {
	srv := Prometheus(PrometheusOpts{Port: fmt.Sprintf("%d", s.freePort())})
	s.Require().NotNil(srv)

	defer srv.Shutdown(context.Background())

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	handler, pattern := http.DefaultServeMux.Handler(req)

	s.Empty(pattern, "metrics must not be registered on the default mux")
	s.NotNil(handler)
}

// Regression test for the listener calling log.Fatal: a metrics port that is
// already taken must never bring down the process serving traffic.
func (s *PrometheusSuite) Test_ServerSurvivesPortInUse() {
	port := s.freePort()

	blocker, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	s.Require().NoError(err)

	defer blocker.Close()

	srv := Prometheus(PrometheusOpts{Port: fmt.Sprintf("%d", port)})

	s.Nil(srv, "an explicit port that is taken should fail rather than move")
	s.True(true, "process is still alive")
}

// An explicitly configured port must be bound strictly. Silently moving it
// leaves the operator with a DOWN scrape target against a running container.
func (s *PrometheusSuite) Test_ExplicitPortDoesNotProbe() {
	port := s.freePort()

	blocker, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	s.Require().NoError(err)

	defer blocker.Close()

	listener := &metricsListener{port: fmt.Sprintf("%d", port), probe: false}

	ln, err := listener.listen()

	s.Error(err)
	s.Nil(ln)
}

// Probing is what lets goreman run hosting and workerserver side by side in
// local development. It must keep working.
func (s *PrometheusSuite) Test_DefaultPortProbesWhenBusy() {
	port := s.freePort()

	blocker, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	s.Require().NoError(err)

	defer blocker.Close()

	listener := &metricsListener{port: fmt.Sprintf("%d", port), probe: true}

	ln, err := listener.listen()

	s.Require().NoError(err)
	defer ln.Close()

	s.Equal(port+1, ln.Addr().(*net.TCPAddr).Port)
}

// A port that does not parse must be refused. Falling back to 0 would make
// net.Listen pick an arbitrary free port, which is the silent move that an
// explicitly configured port is supposed to rule out.
func (s *PrometheusSuite) Test_InvalidPortIsRefused() {
	for _, port := range []string{"2112 ", ":2112", "abc", "", "0", "70000", "-1"} {
		listener := &metricsListener{port: port, probe: false}

		ln, err := listener.listen()

		s.Error(err, "port %q should be refused", port)

		if ln != nil {
			ln.Close()
			s.Fail("port %q bound a listener instead of failing", port)
		}
	}
}

func TestPrometheusSuite(t *testing.T) {
	suite.Run(t, new(PrometheusSuite))
}
