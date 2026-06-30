package hosting

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/lib/pool"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

const collectTestUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36"

type CollectHandlerSuite struct {
	suite.Suite

	captured   []*jobs.HostingRecord
	capturedMu sync.Mutex
	host       *Host
}

func (s *CollectHandlerSuite) SetupTest() {
	s.captured = nil

	// A size-1 buffer flushes each pushed record straight into the capture slice,
	// so the test can inspect what WithCollect queued.
	mu.Lock()
	Batcher = pool.New(
		pool.WithSize(1),
		pool.WithFlushInterval(5*time.Millisecond),
		pool.WithFlusher(pool.FlusherFunc(func(items []any) {
			s.capturedMu.Lock()
			defer s.capturedMu.Unlock()

			for _, it := range items {
				if rec, ok := it.(*jobs.HostingRecord); ok {
					s.captured = append(s.captured, rec)
				}
			}
		})),
	)
	mu.Unlock()

	s.host = &Host{
		Name: "www.stormkit.io",
		Config: &appconf.Config{
			IsEnterprise: true,
			AppID:        types.ID(1),
			EnvID:        types.ID(2),
			DomainID:     types.ID(3),
		},
	}
}

// TearDownSuite clears the global Batcher this suite swapped in, so a later
// suite that relies on the lazy Redis-backed default (e.g. BatcherSuite) gets a
// clean nil to initialize from.
func (s *CollectHandlerSuite) TearDownSuite() {
	mu.Lock()
	Batcher = nil
	mu.Unlock()
}

// request builds a collect request from a unique source IP, so the per-IP rate
// limiter does not bleed budget across tests.
func (s *CollectHandlerSuite) request(method, ip, ua, body string) *RequestContext {
	h := make(http.Header)

	if ua != "" {
		h.Set("User-Agent", ua)
	}

	req := &http.Request{
		Method:     method,
		Header:     h,
		URL:        &url.URL{Path: collectPath},
		RemoteAddr: ip + ":40000",
	}

	if body != "" {
		req.Body = io.NopCloser(strings.NewReader(body))
	}

	return &RequestContext{
		Host:           s.host,
		RequestContext: shttp.NewRequestContext(req),
	}
}

func (s *CollectHandlerSuite) waitForRecords(n int) []*jobs.HostingRecord {
	deadline := time.Now().Add(time.Second)

	for time.Now().Before(deadline) {
		s.capturedMu.Lock()
		got := len(s.captured)
		s.capturedMu.Unlock()

		if got >= n {
			break
		}

		time.Sleep(5 * time.Millisecond)
	}

	s.capturedMu.Lock()
	defer s.capturedMu.Unlock()

	out := make([]*jobs.HostingRecord, len(s.captured))
	copy(out, s.captured)

	return out
}

func (s *CollectHandlerSuite) Test_NonEnterprise_NotFound() {
	s.host.Config.IsEnterprise = false

	res, err := WithCollect(s.request(http.MethodPost, "203.0.113.1", collectTestUA, `{"events":[{"name":"x"}]}`))

	s.NoError(err)
	s.Equal(http.StatusNotFound, res.Status)
}

func (s *CollectHandlerSuite) Test_MethodNotAllowed() {
	res, err := WithCollect(s.request(http.MethodGet, "203.0.113.2", collectTestUA, ""))

	s.NoError(err)
	s.Equal(http.StatusMethodNotAllowed, res.Status)
}

func (s *CollectHandlerSuite) Test_Bot_Dropped() {
	res, err := WithCollect(s.request(http.MethodPost, "203.0.113.3", "Googlebot/2.1", `{"events":[{"name":"x"}]}`))

	s.NoError(err)
	s.Equal(http.StatusNoContent, res.Status)
	s.Empty(s.waitForRecords(1))
}

func (s *CollectHandlerSuite) Test_BadBody() {
	res, err := WithCollect(s.request(http.MethodPost, "203.0.113.4", collectTestUA, `not-json`))

	s.NoError(err)
	s.Equal(http.StatusBadRequest, res.Status)
}

func (s *CollectHandlerSuite) Test_EmptyPayload() {
	res, err := WithCollect(s.request(http.MethodPost, "203.0.113.5", collectTestUA, `{"events":[],"pageviews":[]}`))

	s.NoError(err)
	s.Equal(http.StatusBadRequest, res.Status)
}

func (s *CollectHandlerSuite) Test_Events_Queued() {
	body := `{"events":[{"name":"purchase","path":"/checkout","metadata":{"plan":"pro"}},{"name":""}]}`

	res, err := WithCollect(s.request(http.MethodPost, "203.0.113.6", collectTestUA, body))

	s.NoError(err)
	s.Equal(http.StatusNoContent, res.Status)

	records := s.waitForRecords(1)
	s.Require().Len(records, 1)
	s.Require().Len(records[0].Events, 1) // the empty-name event is dropped

	ev := records[0].Events[0]
	s.Equal("purchase", ev.EventName)
	s.Equal("/checkout", ev.RequestPath.ValueOrZero())
	s.True(ev.VisitorID.Valid)
	s.JSONEq(`{"plan":"pro"}`, ev.Metadata.ValueOrZero())
}

func (s *CollectHandlerSuite) Test_Pageviews_TaggedClient() {
	body := `{"pageviews":[{"path":"/pricing","referrer":"https://google.com"},{"path":""}]}`

	res, err := WithCollect(s.request(http.MethodPost, "203.0.113.7", collectTestUA, body))

	s.NoError(err)
	s.Equal(http.StatusNoContent, res.Status)

	records := s.waitForRecords(1)
	s.Require().Len(records, 1) // the empty-path pageview is dropped
	s.Require().NotNil(records[0].Analytics)

	s.Equal("/pricing", records[0].Analytics.RequestPath)
	s.Equal("client", records[0].Analytics.Source.ValueOrZero())
	s.Equal("203.0.113.7", records[0].Analytics.VisitorIP)
}

func (s *CollectHandlerSuite) Test_Events_Truncated() {
	var b strings.Builder
	b.WriteString(`{"events":[`)

	for i := 0; i < maxEventsPerBeacon+10; i++ {
		if i > 0 {
			b.WriteString(",")
		}

		b.WriteString(`{"name":"e"}`)
	}

	b.WriteString(`]}`)

	res, err := WithCollect(s.request(http.MethodPost, "203.0.113.8", collectTestUA, b.String()))

	s.NoError(err)
	s.Equal(http.StatusNoContent, res.Status)

	records := s.waitForRecords(1)
	s.Require().Len(records, 1)
	s.Len(records[0].Events, maxEventsPerBeacon)
}

func (s *CollectHandlerSuite) Test_RateLimited() {
	body := `{"pageviews":[{"path":"/"}]}`
	statuses := make([]int, 0, 40)

	for i := 0; i < 40; i++ {
		res, err := WithCollect(s.request(http.MethodPost, "198.51.100.42", collectTestUA, body))
		s.NoError(err)
		statuses = append(statuses, res.Status)
	}

	s.Equal(http.StatusNoContent, statuses[0])
	s.Contains(statuses, http.StatusTooManyRequests)
}

func TestCollectHandlerSuite(t *testing.T) {
	suite.Run(t, new(CollectHandlerSuite))
}
