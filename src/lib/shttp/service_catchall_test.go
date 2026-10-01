package shttp_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/tracking"
	"github.com/stretchr/testify/suite"
)

// slowClientWriter stands in for a client that drains the body slowly: every
// write to it takes writeDelay, the way a write to an HTTP/2 stream blocks on
// flow control until the peer has room.
type slowClientWriter struct {
	*httptest.ResponseRecorder
	writeDelay time.Duration
}

func (w *slowClientWriter) Write(b []byte) (int, error) {
	time.Sleep(w.writeDelay)
	return w.ResponseRecorder.Write(b)
}

// sample is one reading of a histogram: how many observations it holds and
// their sum in milliseconds.
type sample struct {
	count uint64
	sumMs float64
}

type CatchAllResponseTimeSuite struct {
	suite.Suite

	restoreConfig func()
}

func (s *CatchAllResponseTimeSuite) SetupTest() {
	cnf := config.Get()
	previous := cnf.Tracking
	cnf.Tracking = &config.TrackingConfig{Prometheus: true}

	s.restoreConfig = func() { cnf.Tracking = previous }
}

func (s *CatchAllResponseTimeSuite) TearDownTest() {
	s.restoreConfig()
}

// handler builds the hosting middleware stack around a catch-all that answers
// with the given response: gzip inside, request timing outside, as main.go
// registers them.
func (s *CatchAllResponseTimeSuite) handler(fn shttp.RequestFunc, devDomain string) http.Handler {
	router := shttp.NewRouter()

	router.RegisterService(func(r *shttp.Router) *shttp.Service {
		svc := r.NewService()
		svc.NewEndpoint("/").CatchAll(fn, devDomain)

		return svc
	})

	return router.WithGzip().WithRequestTiming().Handler()
}

// read returns the GET 200 series of a histogram straight from the collector.
func (s *CatchAllResponseTimeSuite) read(vec *prometheus.HistogramVec) sample {
	metric := &dto.Metric{}

	err := vec.WithLabelValues(http.MethodGet, "200").(interface {
		Write(*dto.Metric) error
	}).Write(metric)
	s.Require().NoError(err)

	return sample{
		count: metric.GetHistogram().GetSampleCount(),
		sumMs: metric.GetHistogram().GetSampleSum(),
	}
}

// Test_ResponseTimeExcludesClientTransfer verifies that the time a slow client
// takes to receive the body does not count toward the response time. That
// metric must describe server work, whichever way the body happens to be
// written.
func (s *CatchAllResponseTimeSuite) Test_ResponseTimeExcludesClientTransfer() {
	const writeDelay = 150 * time.Millisecond

	body := bytes.Repeat([]byte("a"), 64<<10)
	handler := s.handler(func(req *shttp.RequestContext) *shttp.Response {
		return &shttp.Response{
			Status:  http.StatusOK,
			Headers: http.Header{"Content-Type": {"text/plain"}},
			Data:    body,
		}
	}, "")

	before := s.read(tracking.RTHistogramProdEndpoints)

	rec := &slowClientWriter{ResponseRecorder: httptest.NewRecorder(), writeDelay: writeDelay}
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://example.org/asset.js", nil))

	s.Equal(http.StatusOK, rec.Code)
	s.Equal(len(body), rec.Body.Len())

	after := s.read(tracking.RTHistogramProdEndpoints)
	s.Equal(before.count+1, after.count)
	s.Less(after.sumMs-before.sumMs, float64(writeDelay.Milliseconds()), "response time includes the client transfer")
}

// Test_TotalTimeIncludesClientTransfer verifies that the end-to-end histogram
// does include the transfer, even when the body goes through the gzip
// middleware and is flushed only after the handler returns.
func (s *CatchAllResponseTimeSuite) Test_TotalTimeIncludesClientTransfer() {
	const writeDelay = 150 * time.Millisecond

	body := bytes.Repeat([]byte("a"), 64<<10)
	handler := s.handler(func(req *shttp.RequestContext) *shttp.Response {
		return &shttp.Response{
			Status:  http.StatusOK,
			Headers: http.Header{"Content-Type": {"text/plain"}},
			Data:    body,
		}
	}, "")

	before := s.read(tracking.RTTotalHistogramProdEndpoints)

	rec := &slowClientWriter{ResponseRecorder: httptest.NewRecorder(), writeDelay: writeDelay}
	req := httptest.NewRequest(http.MethodGet, "http://example.org/asset.js", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	handler.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.Equal("gzip", rec.Header().Get("Content-Encoding"))

	after := s.read(tracking.RTTotalHistogramProdEndpoints)
	s.Equal(before.count+1, after.count)
	s.GreaterOrEqual(after.sumMs-before.sumMs, float64(writeDelay.Milliseconds()), "total time misses the client transfer")
}

// Test_IncludesHandlerWork verifies that time spent producing the response is
// measured by both histograms.
func (s *CatchAllResponseTimeSuite) Test_IncludesHandlerWork() {
	const work = 40 * time.Millisecond

	handler := s.handler(func(req *shttp.RequestContext) *shttp.Response {
		time.Sleep(work)
		return shttp.OK()
	}, "")

	responseBefore := s.read(tracking.RTHistogramProdEndpoints)
	totalBefore := s.read(tracking.RTTotalHistogramProdEndpoints)

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "http://example.org/", nil))

	responseAfter := s.read(tracking.RTHistogramProdEndpoints)
	totalAfter := s.read(tracking.RTTotalHistogramProdEndpoints)

	s.Equal(responseBefore.count+1, responseAfter.count)
	s.GreaterOrEqual(responseAfter.sumMs-responseBefore.sumMs, float64(work.Milliseconds()))

	s.Equal(totalBefore.count+1, totalAfter.count)
	s.GreaterOrEqual(totalAfter.sumMs-totalBefore.sumMs, float64(work.Milliseconds()))
}

// Test_SkipsDevDomain verifies that requests to the dev domain are recorded by
// neither histogram.
func (s *CatchAllResponseTimeSuite) Test_SkipsDevDomain() {
	handler := s.handler(func(req *shttp.RequestContext) *shttp.Response {
		return shttp.OK()
	}, "stormkit.dev")

	responseBefore := s.read(tracking.RTHistogramProdEndpoints)
	totalBefore := s.read(tracking.RTTotalHistogramProdEndpoints)

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "http://my-app.stormkit.dev/", nil))

	s.Equal(responseBefore.count, s.read(tracking.RTHistogramProdEndpoints).count)
	s.Equal(totalBefore.count, s.read(tracking.RTTotalHistogramProdEndpoints).count)
}

func TestCatchAllResponseTimeSuite(t *testing.T) {
	suite.Run(t, new(CatchAllResponseTimeSuite))
}
