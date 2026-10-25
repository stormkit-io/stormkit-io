package hosting

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
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

// TearDownTest closes the test's Batcher, which waits until every record it
// accepted has been flushed. Otherwise a flush still in flight lands in the
// next test's capture slice and breaks its record counts.
func (s *CollectHandlerSuite) TearDownTest() {
	mu.Lock()
	defer mu.Unlock()

	if Batcher != nil {
		s.Require().NoError(Batcher.Close())
		Batcher = nil
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

	res := handleCollect(s.request(http.MethodPost, "203.0.113.1", collectTestUA, `{"events":[{"name":"x"}]}`))

	s.Equal(http.StatusNotFound, res.Status)
}

func (s *CollectHandlerSuite) Test_MethodNotAllowed() {
	res := handleCollect(s.request(http.MethodGet, "203.0.113.2", collectTestUA, ""))

	s.Equal(http.StatusMethodNotAllowed, res.Status)
}

func (s *CollectHandlerSuite) Test_Bot_Dropped() {
	res := handleCollect(s.request(http.MethodPost, "203.0.113.3", "Googlebot/2.1", `{"events":[{"name":"x"}]}`))

	s.Equal(http.StatusNoContent, res.Status)
	s.Empty(s.waitForRecords(1))
}

// A domain excluded from analytics drops both pageviews and custom events, and
// still answers 204 so a cached script does not surface errors to the visitor.
func (s *CollectHandlerSuite) Test_ExcludedDomain_Dropped() {
	s.host.Config.AnalyticsExcluded = true

	body := `{"events":[{"name":"purchase"}],"pageviews":[{"path":"/pricing"}]}`
	res := handleCollect(s.request(http.MethodPost, "203.0.113.9", collectTestUA, body))

	s.Equal(http.StatusNoContent, res.Status)
	s.Empty(s.waitForRecords(1))
}

func (s *CollectHandlerSuite) Test_BadBody() {
	res := handleCollect(s.request(http.MethodPost, "203.0.113.4", collectTestUA, `not-json`))

	s.Equal(http.StatusBadRequest, res.Status)
}

func (s *CollectHandlerSuite) Test_EmptyPayload() {
	res := handleCollect(s.request(http.MethodPost, "203.0.113.5", collectTestUA, `{"events":[],"pageviews":[]}`))

	s.Equal(http.StatusBadRequest, res.Status)
}

func (s *CollectHandlerSuite) Test_Events_Queued() {
	body := `{"events":[{"name":"purchase","path":"/checkout","metadata":{"plan":"pro"}},{"name":""}]}`

	res := handleCollect(s.request(http.MethodPost, "203.0.113.6", collectTestUA, body))

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

	res := handleCollect(s.request(http.MethodPost, "203.0.113.7", collectTestUA, body))

	s.Equal(http.StatusNoContent, res.Status)

	records := s.waitForRecords(1)
	s.Require().Len(records, 1) // the empty-path pageview is dropped
	s.Require().NotNil(records[0].Analytics)

	s.Equal("/pricing", records[0].Analytics.RequestPath)
	s.Equal("client", records[0].Analytics.Source.ValueOrZero())
	s.Equal("203.0.113.7", records[0].Analytics.VisitorIP)
}

// Test_CrossOriginBeacon_Dropped verifies that a page on another site cannot
// write page views into this site's analytics through visitors' browsers.
func (s *CollectHandlerSuite) Test_CrossOriginBeacon_Dropped() {
	body := `{"pageviews":[{"path":"/spam"}]}`
	headers := []map[string]string{
		{"Sec-Fetch-Site": "cross-site"},
		{"Sec-Fetch-Site": "same-site"},
		{"Origin": "https://evil.example"},
		{"Origin": "null"},
	}

	for i, h := range headers {
		req := s.request(http.MethodPost, "203.0.113."+strconv.Itoa(20+i), collectTestUA, body)

		for k, v := range h {
			req.Header.Set(k, v)
		}

		s.Equal(http.StatusNoContent, handleCollect(req).Status)
	}

	s.Empty(s.waitForRecords(1))
}

func (s *CollectHandlerSuite) Test_SameOriginBeacon_Recorded() {
	body := `{"pageviews":[{"path":"/pricing"}]}`
	headers := []map[string]string{
		{"Sec-Fetch-Site": "same-origin", "Origin": "https://www.stormkit.io"},
		{"Origin": "https://WWW.stormkit.io"},
		{}, // server-to-server
	}

	for i, h := range headers {
		req := s.request(http.MethodPost, "203.0.113."+strconv.Itoa(30+i), collectTestUA, body)

		for k, v := range h {
			req.Header.Set(k, v)
		}

		s.Equal(http.StatusNoContent, handleCollect(req).Status)
	}

	s.Len(s.waitForRecords(len(headers)), len(headers))
}

// Test_Pageviews_Campaign verifies that a landing the server never saw keeps
// the campaign from the query string the script sends.
func (s *CollectHandlerSuite) Test_Pageviews_Campaign() {
	body := `{"pageviews":[{"path":"/launch","search":"?utm_source=x&ref=hn"},{"path":"/about"}]}`

	res := handleCollect(s.request(http.MethodPost, "203.0.113.40", collectTestUA, body))

	s.Equal(http.StatusNoContent, res.Status)

	records := s.waitForRecords(2)
	s.Require().Len(records, 2)

	byPath := map[string]*jobs.HostingRecord{}

	for _, r := range records {
		byPath[r.Analytics.RequestPath] = r
	}

	s.Equal("x", byPath["/launch"].Analytics.Campaign.Source.ValueOrZero())
	s.Equal("hn", byPath["/launch"].Analytics.Campaign.Ref.ValueOrZero())
	s.False(byPath["/about"].Analytics.Campaign.Source.Valid)
}

// Test_Engagements_Queued verifies that foreground time reports are queued with
// their page view id, capped, and dropped when incomplete.
func (s *CollectHandlerSuite) Test_Engagements_Queued() {
	body := `{"engagements":[
		{"path":"/blog","viewId":"v1","seconds":42},
		{"path":"/blog","viewId":"v2","seconds":999999},
		{"path":"","viewId":"v3","seconds":5},
		{"path":"/blog","viewId":"","seconds":5},
		{"path":"/blog","viewId":"v4","seconds":0},
		{"path":"/blog","viewId":"` + strings.Repeat("x", 65) + `","seconds":5}
	]}`

	res := handleCollect(s.request(http.MethodPost, "203.0.113.50", collectTestUA, body))

	s.Equal(http.StatusNoContent, res.Status)

	records := s.waitForRecords(1)
	s.Require().Len(records, 1)
	s.Require().Len(records[0].Engagements, 2)

	s.Equal("v1", records[0].Engagements[0].ViewID)
	s.Equal(42, records[0].Engagements[0].Seconds)
	s.Equal(analytics.MaxEngagementSeconds, records[0].Engagements[1].Seconds)
	s.Equal(types.ID(3), records[0].Engagements[0].DomainID)
}

// Test_Engagements_MergedPerView verifies that repeating a view id in one
// beacon cannot stack the per-report cap.
func (s *CollectHandlerSuite) Test_Engagements_MergedPerView() {
	var b strings.Builder
	b.WriteString(`{"engagements":[`)

	for i := 0; i < maxEventsPerBeacon; i++ {
		if i > 0 {
			b.WriteString(",")
		}

		b.WriteString(`{"path":"/blog","viewId":"v1","seconds":1800}`)
	}

	b.WriteString(`,{"path":"/blog","viewId":"v2","seconds":10}]}`)

	res := handleCollect(s.request(http.MethodPost, "203.0.113.51", collectTestUA, b.String()))

	s.Equal(http.StatusNoContent, res.Status)

	records := s.waitForRecords(1)
	s.Require().Len(records, 1)
	s.Require().Len(records[0].Engagements, 1, "v2 is past the per-beacon item cap")
	s.Equal("v1", records[0].Engagements[0].ViewID)
	s.Equal(analytics.MaxEngagementSeconds, records[0].Engagements[0].Seconds)
}

func (s *CollectHandlerSuite) Test_Engagements_SumsReportsOfOneView() {
	body := `{"engagements":[
		{"path":"/blog","viewId":"v1","seconds":10},
		{"path":"/blog","viewId":"v2","seconds":7},
		{"path":"/blog","viewId":"v1","seconds":15}
	]}`

	res := handleCollect(s.request(http.MethodPost, "203.0.113.52", collectTestUA, body))

	s.Equal(http.StatusNoContent, res.Status)

	records := s.waitForRecords(1)
	s.Require().Len(records, 1)
	s.Require().Len(records[0].Engagements, 2)
	s.Equal(25, records[0].Engagements[0].Seconds)
	s.Equal(7, records[0].Engagements[1].Seconds)
}

// Test_LongPaths_Dropped verifies that items with an absurdly long path are
// dropped instead of being stored.
func (s *CollectHandlerSuite) Test_LongPaths_Dropped() {
	long := "/" + strings.Repeat("a", maxBeaconPathLength)
	body := `{
		"pageviews":[{"path":"` + long + `"},{"path":"/ok"}],
		"events":[{"name":"x","path":"` + long + `"}],
		"engagements":[{"path":"` + long + `","viewId":"v1","seconds":3}]
	}`

	res := handleCollect(s.request(http.MethodPost, "203.0.113.61", collectTestUA, body))

	s.Equal(http.StatusNoContent, res.Status)

	records := s.waitForRecords(2)
	s.Require().Len(records, 1)
	s.Require().NotNil(records[0].Analytics)
	s.Equal("/ok", records[0].Analytics.RequestPath)
}

// Test_Events_Sanitized verifies that a long name is truncated and that poison
// or oversized metadata is dropped while the event is kept.
func (s *CollectHandlerSuite) Test_Events_Sanitized() {
	body := `{"events":[
		{"name":"` + strings.Repeat("n", maxEventNameLength+10) + `"},
		{"name":"surrogate","metadata":{"a":"\ud800"}},
		{"name":"big","metadata":{"a":"` + strings.Repeat("x", maxMetadataLength) + `"}}
	]}`

	res := handleCollect(s.request(http.MethodPost, "203.0.113.62", collectTestUA, body))

	s.Equal(http.StatusNoContent, res.Status)

	records := s.waitForRecords(1)
	s.Require().Len(records, 1)
	s.Require().Len(records[0].Events, 3)

	s.Equal(strings.Repeat("n", maxEventNameLength), records[0].Events[0].EventName)
	s.JSONEq(`{"a":"\ufffd"}`, records[0].Events[1].Metadata.ValueOrZero())
	s.False(records[0].Events[2].Metadata.Valid)
}

// Test_NoDomain_Dropped verifies that beacons from a host without a domain are
// not queued, since analytics are only ever read per domain.
func (s *CollectHandlerSuite) Test_NoDomain_Dropped() {
	config.SetIsSelfHosted(true)
	defer config.SetIsSelfHosted(false)

	s.host.Config.DomainID = 0

	body := `{"events":[{"name":"purchase"}],"pageviews":[{"path":"/pricing"}]}`
	res := handleCollect(s.request(http.MethodPost, "203.0.113.63", collectTestUA, body))

	s.Equal(http.StatusNoContent, res.Status)
	s.Empty(s.waitForRecords(1))
}

// Test_PathsDecoded verifies that beacon paths are decoded like the server
// decodes request paths, so client and server rows for one page group together.
func (s *CollectHandlerSuite) Test_PathsDecoded() {
	body := `{
		"pageviews":[{"path":"/caf%C3%A9"}],
		"events":[{"name":"x","path":"/caf%C3%A9"}],
		"engagements":[{"path":"/caf%C3%A9","viewId":"v1","seconds":3},{"path":"/100%","viewId":"v2","seconds":3}]
	}`

	res := handleCollect(s.request(http.MethodPost, "203.0.113.60", collectTestUA, body))

	s.Equal(http.StatusNoContent, res.Status)

	paths := []string{}

	for _, r := range s.waitForRecords(3) {
		if r.Analytics != nil {
			paths = append(paths, r.Analytics.RequestPath)
		}

		for _, e := range r.Events {
			paths = append(paths, e.RequestPath.ValueOrZero())
		}

		for _, e := range r.Engagements {
			paths = append(paths, e.RequestPath)
		}
	}

	s.ElementsMatch([]string{"/café", "/café", "/café", "/100%"}, paths, "a malformed encoding is kept as sent")
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

	res := handleCollect(s.request(http.MethodPost, "203.0.113.8", collectTestUA, b.String()))

	s.Equal(http.StatusNoContent, res.Status)

	records := s.waitForRecords(1)
	s.Require().Len(records, 1)
	s.Len(records[0].Events, maxEventsPerBeacon)
}

func (s *CollectHandlerSuite) Test_RateLimited() {
	body := `{"pageviews":[{"path":"/"}]}`
	statuses := make([]int, 0, 40)

	for i := 0; i < 40; i++ {
		res := handleCollect(s.request(http.MethodPost, "198.51.100.42", collectTestUA, body))
		statuses = append(statuses, res.Status)
	}

	s.Equal(http.StatusNoContent, statuses[0])
	s.Contains(statuses, http.StatusTooManyRequests)
}

func TestCollectHandlerSuite(t *testing.T) {
	suite.Run(t, new(CollectHandlerSuite))
}
