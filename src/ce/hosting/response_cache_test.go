package hosting_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/pool"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type ResponseCacheSuite struct {
	suite.Suite

	mockClient *mocks.ClientInterface
	records    []*jobs.HostingRecord
	recordsMu  sync.Mutex
}

type cacheRequestParams struct {
	Host      *hosting.Host
	Path      string
	Method    string
	Headers   map[string]string
	RequestID string
}

type cacheFileParams struct {
	Name        string
	ContentType string
	Content     []byte
}

func (s *ResponseCacheSuite) BeforeTest(_, _ string) {
	hosting.WaitArtifacts()

	// The previous test's flusher can still be running.
	s.recordsMu.Lock()
	s.records = nil
	s.recordsMu.Unlock()

	hosting.ResetBatcher(pool.New(
		pool.WithSize(1),
		pool.WithFlushInterval(5*time.Millisecond),
		pool.WithFlusher(pool.FlusherFunc(func(items []any) {
			s.recordsMu.Lock()
			defer s.recordsMu.Unlock()

			for _, item := range items {
				if record, ok := item.(*jobs.HostingRecord); ok {
					s.records = append(s.records, record)
				}
			}
		})),
	))

	hosting.ResetResponseCache(64 << 20)

	s.mockClient = &mocks.ClientInterface{}
	integrations.SetDefaultClient(s.mockClient)
	admin.SetMockLicense()
}

func (s *ResponseCacheSuite) AfterTest(_, _ string) {
	hosting.WaitArtifacts()
	admin.ResetMockLicense()
}

func (s *ResponseCacheSuite) TearDownSuite() {
	integrations.SetDefaultClient(nil)
}

func (s *ResponseCacheSuite) page(text string) []byte {
	return []byte("<html><head></head><body>" + strings.Repeat("<p>"+text+"</p>\n", 200) + "</body></html>")
}

// host returns a host with a fresh config, the way a reload produces one.
func (s *ResponseCacheSuite) host(files ...string) *hosting.Host {
	static := appconf.StaticFileConfig{}

	// The build manifest carries each file's type; without one the handler
	// defaults to HTML.
	for _, name := range files {
		static[name] = &appconf.StaticFile{
			FileName: name,
			Headers:  map[string]string{"Content-Type": mime.TypeByExtension(path.Ext(name))},
		}
	}

	return &hosting.Host{
		Name: "www.stormkit.io",
		Config: &appconf.Config{
			DeploymentID:    types.ID(1),
			StorageLocation: "aws:my-bucket/my-key-prefix",
			StaticFiles:     static,
		},
	}
}

func (s *ResponseCacheSuite) mockFile(p cacheFileParams) {
	s.mockClient.On("GetFile", integrations.GetFileArgs{
		Location:     "aws:my-bucket/my-key-prefix",
		FileName:     p.Name,
		DeploymentID: types.ID(1),
	}).Return(&integrations.GetFileResult{
		Content:     p.Content,
		ContentType: p.ContentType,
		Size:        int64(len(p.Content)),
	}, nil)
}

func (s *ResponseCacheSuite) mockPage(name string, content []byte) {
	s.mockFile(cacheFileParams{Name: name, ContentType: "text/html; charset=utf-8", Content: content})
}

func (s *ResponseCacheSuite) serve(p cacheRequestParams) *shttp.Response {
	headers := http.Header{}

	for key, value := range p.Headers {
		headers.Set(key, value)
	}

	requestPath, query, _ := strings.Cut(p.Path, "?")

	rq := &hosting.RequestContext{
		Host:      p.Host,
		RequestID: p.RequestID,
		RequestContext: shttp.NewRequestContext(&http.Request{
			Method: p.Method,
			Header: headers,
			URL:    &url.URL{Host: p.Host.Name, Path: requestPath, RawQuery: query},
		}),
	}

	rq.OriginalPath = requestPath

	return hosting.HandlerForward(rq)
}

func (s *ResponseCacheSuite) gzipRequest(host *hosting.Host, path string) *shttp.Response {
	return s.serve(cacheRequestParams{Host: host, Path: path, Headers: map[string]string{"Accept-Encoding": "gzip, deflate, br"}})
}

// body returns what the client reads, decoding a compressed body.
func (s *ResponseCacheSuite) body(res *shttp.Response) string {
	var raw []byte

	switch data := res.Data.(type) {
	case []byte:
		raw = data
	case string:
		raw = []byte(data)
	}

	if res.Headers.Get("Content-Encoding") != "gzip" {
		return string(raw)
	}

	reader, err := gzip.NewReader(bytes.NewReader(raw))
	s.Require().NoError(err)

	decoded, err := io.ReadAll(reader)
	s.Require().NoError(err)

	return string(decoded)
}

func (s *ResponseCacheSuite) Test_RepeatRequestIsServedFromCache() {
	page := s.page("hello")
	host := s.host("/page.html")

	s.mockPage("/page.html", page)

	first := s.gzipRequest(host, "/page.html")
	second := s.gzipRequest(host, "/page.html")

	for _, res := range []*shttp.Response{first, second} {
		s.Equal(http.StatusOK, res.Status)
		s.Equal("gzip", res.Headers.Get("Content-Encoding"))
		s.Contains(res.Headers.Get("Vary"), "Accept-Encoding")
		s.Equal("1", res.Headers.Get("x-sk-version"), "the cached copy carries the finalized headers")
		s.Equal(string(page), s.body(res))
	}

	s.mockClient.AssertNumberOfCalls(s.T(), "GetFile", 1)
	s.Equal(1, hosting.ResponseCacheLen())
}

func (s *ResponseCacheSuite) Test_ClientWithoutGzipGetsPlainBody() {
	page := s.page("hello")
	host := s.host("/page.html")

	s.mockPage("/page.html", page)

	for _, acceptEncoding := range []string{"", "deflate", "gzip;q=0", "gzip;q=0.0", "gzip; q=0.000", "gzip;q=abc"} {
		res := s.serve(cacheRequestParams{Host: host, Path: "/page.html", Headers: map[string]string{"Accept-Encoding": acceptEncoding}})

		s.Empty(res.Headers.Get("Content-Encoding"), "Accept-Encoding %q must not get a compressed body", acceptEncoding)
		s.Equal(string(page), s.body(res))
	}

	s.Equal(0, hosting.ResponseCacheLen(), "a client that cannot be served the copy must not pay to build it")

	res := s.serve(cacheRequestParams{Host: host, Path: "/page.html", Headers: map[string]string{"Accept-Encoding": "gzip;q=0.5"}})

	s.Equal("gzip", res.Headers.Get("Content-Encoding"))
	s.Equal(1, hosting.ResponseCacheLen())
	s.mockClient.AssertNumberOfCalls(s.T(), "GetFile", 7)
}

func (s *ResponseCacheSuite) Test_SnippetsAreCachedInjected() {
	host := s.host("/page.html")
	host.Config.Snippets = appconf.Snippets{{Content: "<!--sk-->", Location: "head"}}

	s.mockPage("/page.html", s.page("hello"))

	for range 2 {
		res := s.gzipRequest(host, "/page.html")

		s.Equal("gzip", res.Headers.Get("Content-Encoding"))
		s.Contains(s.body(res), "<!--sk--></head>")
	}

	s.mockClient.AssertNumberOfCalls(s.T(), "GetFile", 1)
}

func (s *ResponseCacheSuite) Test_RequestIDSnippetIsNeverCached() {
	host := s.host("/page.html")
	host.Config.Snippets = appconf.Snippets{{Content: "<!--" + appconf.SnippetVarRequestID + "-->", Location: "head", Interpolate: true}}

	s.mockPage("/page.html", s.page("hello"))

	for _, id := range []string{"first", "second"} {
		res := s.serve(cacheRequestParams{
			Host:      host,
			Path:      "/page.html",
			RequestID: id,
			Headers:   map[string]string{"Accept-Encoding": "gzip"},
		})

		s.Contains(s.body(res), "<!--"+id+"-->")
	}

	s.mockClient.AssertNumberOfCalls(s.T(), "GetFile", 2)
	s.Equal(0, hosting.ResponseCacheLen())
}

func (s *ResponseCacheSuite) Test_ReloadedConfigMisses() {
	s.mockPage("/page.html", s.page("hello"))

	before := s.host("/page.html")
	before.Config.Snippets = appconf.Snippets{{Content: "<!--old-->", Location: "head"}}

	s.Contains(s.body(s.gzipRequest(before, "/page.html")), "<!--old-->")

	after := s.host("/page.html")
	after.Config.Snippets = appconf.Snippets{{Content: "<!--new-->", Location: "head"}}

	body := s.body(s.gzipRequest(after, "/page.html"))

	s.Contains(body, "<!--new-->")
	s.NotContains(body, "<!--old-->")
	s.mockClient.AssertNumberOfCalls(s.T(), "GetFile", 2)
}

func (s *ResponseCacheSuite) Test_InvalidationDropsMatchingHosts() {
	s.mockPage("/page.html", s.page("hello"))

	www := s.host("/page.html")
	other := s.host("/page.html")
	other.Name = "app.example.org"

	s.gzipRequest(www, "/page.html")
	s.gzipRequest(other, "/page.html")
	s.Equal(2, hosting.ResponseCacheLen())

	hosting.InvalidateCache(context.Background(), `^www\.stormkit\.io$`)

	s.Equal(1, hosting.ResponseCacheLen())
}

func (s *ResponseCacheSuite) Test_MarkdownIsCachedApartFromPage() {
	host := s.host("/docs.html", "/docs.md")
	host.Config.Markdown = true

	page := s.page("page")
	markdown := []byte(strings.Repeat("# markdown\n\nSome text.\n", 100))

	s.mockPage("/docs.html", page)
	s.mockFile(cacheFileParams{Name: "/docs.md", ContentType: "text/markdown", Content: markdown})

	for _, accept := range []string{"text/html", "text/markdown", "text/html", "text/markdown"} {
		res := s.serve(cacheRequestParams{
			Host:    host,
			Path:    "/docs",
			Headers: map[string]string{"Accept": accept, "Accept-Encoding": "gzip"},
		})

		if accept == "text/markdown" {
			s.Equal(string(markdown), s.body(res))
		} else {
			s.Equal(string(page), s.body(res))
		}
	}

	s.mockClient.AssertNumberOfCalls(s.T(), "GetFile", 2)
}

func (s *ResponseCacheSuite) Test_ResponsesThatAreNotCached() {
	host := s.host("/page.html", "/logo.png", "/small.html")

	s.mockPage("/page.html", s.page("hello"))
	s.mockPage("/small.html", []byte("<html><head></head><body>hi</body></html>"))
	s.mockFile(cacheFileParams{Name: "/logo.png", ContentType: "image/png", Content: bytes.Repeat([]byte{0x89}, 4096)})

	cases := map[string]cacheRequestParams{
		"not found":        {Path: "/missing"},
		"image resize":     {Path: "/page.html?size=100x100"},
		"conditional":      {Path: "/page.html", Headers: map[string]string{"If-None-Match": `"abc"`}},
		"head request":     {Path: "/page.html", Method: http.MethodHead},
		"image":            {Path: "/logo.png"},
		"body under 1 KiB": {Path: "/small.html"},
	}

	for name, params := range cases {
		s.Run(name, func() {
			params.Host = host

			if params.Headers == nil {
				params.Headers = map[string]string{}
			}

			params.Headers["Accept-Encoding"] = "gzip"

			s.serve(params)
			s.Equal(0, hosting.ResponseCacheLen())
		})
	}
}

func (s *ResponseCacheSuite) Test_BandwidthCountsBodyBeforeCompression() {
	page := s.page("hello")
	host := s.host("/page.html")

	s.mockPage("/page.html", page)

	s.gzipRequest(host, "/page.html")
	s.gzipRequest(host, "/page.html")

	hosting.WaitArtifacts()

	s.Eventually(func() bool {
		s.recordsMu.Lock()
		defer s.recordsMu.Unlock()

		return len(s.records) == 2
	}, time.Second, 5*time.Millisecond)

	s.recordsMu.Lock()
	defer s.recordsMu.Unlock()

	for _, record := range s.records {
		s.GreaterOrEqual(record.TotalBandwidth, int64(len(page)))
	}
}

func (s *ResponseCacheSuite) Test_EvictsLeastRecentlyUsed() {
	page := s.page("hello")
	host := s.host("/a.html", "/b.html")

	s.mockPage("/a.html", page)
	s.mockPage("/b.html", page)

	// Both entries cost the same: equal bodies, equal-length keys.
	s.gzipRequest(host, "/a.html")

	cost := hosting.ResponseCacheUsed()

	// Room for one entry, not two.
	hosting.ResetResponseCache(cost*2 - 1)

	s.gzipRequest(host, "/a.html")
	s.gzipRequest(host, "/b.html")
	s.Equal(1, hosting.ResponseCacheLen())

	s.gzipRequest(host, "/b.html")
	s.mockClient.AssertNumberOfCalls(s.T(), "GetFile", 3)

	s.gzipRequest(host, "/a.html")
	s.mockClient.AssertNumberOfCalls(s.T(), "GetFile", 4)
}

func (s *ResponseCacheSuite) Test_BudgetCountsKeyAndHeaders() {
	page := s.page("hello")
	host := s.host("/page.html")

	s.mockPage("/page.html", page)

	res := s.gzipRequest(host, "/page.html")
	body, ok := res.Data.([]byte)

	s.Require().True(ok)
	s.Greater(hosting.ResponseCacheUsed(), int64(len(body)))
}

func TestResponseCacheSuite(t *testing.T) {
	suite.Run(t, new(ResponseCacheSuite))
}
