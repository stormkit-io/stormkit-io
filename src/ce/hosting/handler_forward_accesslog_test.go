package hosting

import (
	"net/http"
	"net/url"
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

type AccessLogArtifactsSuite struct {
	suite.Suite

	captured   []*jobs.HostingRecord
	capturedMu sync.Mutex
	host       *Host
}

func (s *AccessLogArtifactsSuite) SetupTest() {
	s.captured = nil

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
			AppID:    types.ID(1),
			EnvID:    types.ID(2),
			DomainID: types.ID(3),
		},
	}
}

func (s *AccessLogArtifactsSuite) TearDownSuite() {
	mu.Lock()
	Batcher = nil
	mu.Unlock()
}

func (s *AccessLogArtifactsSuite) waitForRecord() *jobs.HostingRecord {
	deadline := time.Now().Add(time.Second)

	for time.Now().Before(deadline) {
		s.capturedMu.Lock()
		got := len(s.captured)
		s.capturedMu.Unlock()

		if got >= 1 {
			break
		}

		time.Sleep(5 * time.Millisecond)
	}

	s.capturedMu.Lock()
	defer s.capturedMu.Unlock()

	s.Require().NotEmpty(s.captured)

	return s.captured[0]
}

func (s *AccessLogArtifactsSuite) requestServer(ua, path, ip string) *RequestServer {
	h := make(http.Header)
	h.Set("User-Agent", ua)

	req := &http.Request{
		Method:     http.MethodGet,
		Header:     h,
		URL:        &url.URL{Path: path},
		RemoteAddr: ip + ":40000",
	}

	return &RequestServer{
		req: &RequestContext{
			Host:           s.host,
			OriginalPath:   path,
			RequestContext: shttp.NewRequestContext(req),
		},
	}
}

// A bot request must still produce an access log (flagged is_bot), proving the
// raw log keeps bot traffic the analytics path drops.
func (s *AccessLogArtifactsSuite) Test_BotRequest_QueuesAccessLog() {
	rs := s.requestServer("Googlebot/2.1 (+http://www.google.com/bot.html)", "/robots.txt", "198.51.100.9")

	rs.artifacts(&shttp.Response{Status: http.StatusOK, Data: []byte("hello")})

	record := s.waitForRecord()

	s.Require().NotNil(record.AccessLog)
	s.True(record.AccessLog.IsBot)
	s.Equal("/robots.txt", record.AccessLog.RequestPath)
	s.Equal("198.51.100.9", record.AccessLog.ClientIP)
	s.Equal(http.MethodGet, record.AccessLog.Method)
	s.Equal(http.StatusOK, record.AccessLog.StatusCode)
	s.Greater(record.AccessLog.BytesSent, int64(0))
}

func (s *AccessLogArtifactsSuite) Test_HumanRequest_NotFlaggedBot() {
	const ua = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36"

	rs := s.requestServer(ua, "/", "203.0.113.7")

	rs.artifacts(&shttp.Response{Status: http.StatusOK, Data: []byte("ok")})

	record := s.waitForRecord()

	s.Require().NotNil(record.AccessLog)
	s.False(record.AccessLog.IsBot)
	s.Equal("203.0.113.7", record.AccessLog.ClientIP)
}

func TestAccessLogArtifactsSuite(t *testing.T) {
	suite.Run(t, new(AccessLogArtifactsSuite))
}
