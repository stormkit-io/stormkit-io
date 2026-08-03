package sysstats

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

type CollectorSuite struct {
	suite.Suite
	fixture []byte
}

func (s *CollectorSuite) SetupTest() {
	data, err := os.ReadFile("testdata/node_exporter.txt")
	s.Require().NoError(err)
	s.fixture = data
}

func (s *CollectorSuite) server() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/metrics", r.URL.Path)
		_, _ = w.Write(s.fixture)
	}))
}

// advancingServer moves the CPU counters forward on every scrape, the way a
// live exporter does. A static fixture yields a zero time delta, from which no
// rate can be derived.
func (s *CollectorSuite) advancingServer() *httptest.Server {
	idle, busy := 1000.0, 200.0

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idle += 75
		busy += 25

		fmt.Fprintf(w, `# TYPE node_cpu_seconds_total counter
node_cpu_seconds_total{cpu="0",mode="idle"} %f
node_cpu_seconds_total{cpu="0",mode="user"} %f
`, idle, busy)
	}))
}

func (s *CollectorSuite) Test_TargetURL() {
	s.Equal("http://host:9100/metrics", TargetURL("host"))
	s.Equal("http://host:9999/metrics", TargetURL("host:9999"))
	s.Equal("http://host:9100/metrics", TargetURL("http://host:9100"))
	s.Equal("https://host:9100/metrics", TargetURL("https://host:9100/"))
}

func (s *CollectorSuite) Test_Collect_ReturnsSample() {
	srv := s.server()
	defer srv.Close()

	sample := NewCollector(NewCollectorParams{}).Collect(context.Background(), srv.URL)

	s.True(sample.Reachable)
	s.Empty(sample.Error)
	s.Equal(srv.URL, sample.Target)
	s.NotZero(sample.Timestamp)
	s.Len(sample.Filesystems, 2)
}

// A machine that stops answering must still be recorded, so the UI can tell the
// operator node_exporter is not running there.
func (s *CollectorSuite) Test_Collect_RecordsUnreachableTarget() {
	sample := NewCollector(NewCollectorParams{}).Collect(context.Background(), "127.0.0.1:1")

	s.Require().NotNil(sample)
	s.False(sample.Reachable)
	s.NotEmpty(sample.Error)
	s.Equal("127.0.0.1:1", sample.Target)
	s.NotZero(sample.Timestamp)
}

func (s *CollectorSuite) Test_Collect_RecordsNonOKStatus() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	sample := NewCollector(NewCollectorParams{}).Collect(context.Background(), srv.URL)

	s.False(sample.Reachable)
	s.Contains(sample.Error, "500")
}

func (s *CollectorSuite) Test_Collect_RetainsCountersPerTarget() {
	srv := s.advancingServer()
	defer srv.Close()

	c := NewCollector(NewCollectorParams{})

	first := c.Collect(context.Background(), srv.URL)
	s.False(first.CPUValid, "first scrape of a target has no delta")

	second := c.Collect(context.Background(), srv.URL)
	s.True(second.CPUValid, "second scrape derives a rate from the retained counters")
	s.InDelta(25, second.CPUPercent, 0.001)
}

// Two scrapes with no elapsed CPU time cannot produce a rate. Reporting a gap
// is correct here; dividing by a zero delta is not.
func (s *CollectorSuite) Test_Collect_InvalidWhenCountersDoNotAdvance() {
	srv := s.server()
	defer srv.Close()

	c := NewCollector(NewCollectorParams{})
	c.Collect(context.Background(), srv.URL)

	s.False(c.Collect(context.Background(), srv.URL).CPUValid)
}

func (s *CollectorSuite) Test_Forget_ClearsRetainedCounters() {
	srv := s.advancingServer()
	defer srv.Close()

	c := NewCollector(NewCollectorParams{})
	c.Collect(context.Background(), srv.URL)
	c.Forget(srv.URL)

	sample := c.Collect(context.Background(), srv.URL)
	s.False(sample.CPUValid, "a returning machine starts from a clean delta")
}

func TestCollectorSuite(t *testing.T) {
	suite.Run(t, new(CollectorSuite))
}
