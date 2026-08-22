package hosting_test

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/pool"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

// ContentNegotiationSuite covers acceptmarkdown.com compliance: a page that
// ships a markdown twin is served as markdown to clients that ask for it, HTML
// to everyone else, and both answers carry Vary: Accept so a cache cannot mix
// the two up.
type ContentNegotiationSuite struct {
	suite.Suite

	mockClient *mocks.ClientInterface
	tmpDir     string
}

func (s *ContentNegotiationSuite) SetupSuite() {
	tmpDir, err := os.MkdirTemp("", "tmp-test-content-negotiation-")
	s.Require().NoError(err)

	s.tmpDir = tmpDir
}

func (s *ContentNegotiationSuite) TearDownSuite() {
	_ = os.RemoveAll(s.tmpDir)
}

func (s *ContentNegotiationSuite) BeforeTest(_, _ string) {
	rediscache.Client().Del(context.Background(), "content-negotiation")

	hosting.WaitArtifacts()
	hosting.ResetBatcher(pool.New(
		pool.WithSize(1000),
		pool.WithFlushInterval(time.Hour),
		pool.WithFlusher(pool.FlusherFunc(func(items []any) {})),
	))

	s.mockClient = &mocks.ClientInterface{}
	integrations.SetDefaultClient(s.mockClient)
}

func (s *ContentNegotiationSuite) AfterTest(_, _ string) {
	hosting.WaitArtifacts()
}

// host returns a deployment that publishes /docs.html plus its markdown twin.
func (s *ContentNegotiationSuite) host() *hosting.Host {
	return &hosting.Host{
		Name: "www.stormkit.io",
		Config: &appconf.Config{
			DeploymentID:    types.ID(1),
			AppID:           types.ID(25),
			EnvID:           types.ID(100),
			StorageLocation: "aws:my-bucket/my-key-prefix",
			StaticFiles: appconf.StaticFileConfig{
				"/docs.html": {
					FileName: "/docs.html",
					Headers:  map[string]string{"content-type": "text/html; charset=utf-8"},
				},
				"/docs.md": {
					FileName: "/docs.md",
					Headers:  map[string]string{"content-type": "text/markdown; charset=utf-8"},
				},
				"/pricing.html": {
					FileName: "/pricing.html",
					Headers:  map[string]string{"content-type": "text/html; charset=utf-8"},
				},
			},
		},
	}
}

func (s *ContentNegotiationSuite) mockFile(fileName, content string) {
	s.mockClient.On("GetFile", integrations.GetFileArgs{
		Location:     "aws:my-bucket/my-key-prefix",
		FileName:     fileName,
		DeploymentID: types.ID(1),
	}).Return(&integrations.GetFileResult{
		Content:     []byte(content),
		ContentType: "text/html; charset=utf-8",
	}, nil)
}

func (s *ContentNegotiationSuite) request(host *hosting.Host, path, accept string) *shttp.Response {
	headers := make(http.Header)

	if accept != "" {
		headers.Set("Accept", accept)
	}

	req := &hosting.RequestContext{
		Host: host,
		RequestContext: shttp.NewRequestContext(&http.Request{
			Header: headers,
			URL:    &url.URL{Host: host.Name, Path: path, RawPath: path},
		}),
	}

	req.OriginalPath = path

	return hosting.HandlerForward(req)
}

func (s *ContentNegotiationSuite) Test_ServesMarkdownWhenRequested() {
	s.mockFile("/docs.md", "# Docs\n")

	res := s.request(s.host(), "/docs", "text/markdown")

	s.Equal(http.StatusOK, res.Status)
	s.Equal("text/markdown; charset=utf-8", res.Headers.Get("Content-Type"))
	s.Equal("Accept", res.Headers.Get("Vary"))
	s.Equal("# Docs\n", string(res.Data.([]byte)))
}

func (s *ContentNegotiationSuite) Test_ServesHtmlToBrowsers() {
	s.mockFile("/docs.html", "<h1>Docs</h1>")

	res := s.request(s.host(), "/docs", "text/html,application/xhtml+xml,*/*;q=0.8")

	s.Equal(http.StatusOK, res.Status)
	s.True(strings.HasPrefix(res.Headers.Get("Content-Type"), "text/html"))
	s.Equal("Accept", res.Headers.Get("Vary"))
}

// A wildcard-only Accept must not flip the page to markdown: `*/*` is what
// curl and most crawlers send, and they expect the HTML representation.
func (s *ContentNegotiationSuite) Test_WildcardKeepsHtml() {
	s.mockFile("/docs.html", "<h1>Docs</h1>")

	res := s.request(s.host(), "/docs", "*/*")

	s.Equal(http.StatusOK, res.Status)
	s.True(strings.HasPrefix(res.Headers.Get("Content-Type"), "text/html"))
}

func (s *ContentNegotiationSuite) Test_NoAcceptHeaderKeepsHtml() {
	s.mockFile("/docs.html", "<h1>Docs</h1>")

	res := s.request(s.host(), "/docs", "")

	s.Equal(http.StatusOK, res.Status)
	s.True(strings.HasPrefix(res.Headers.Get("Content-Type"), "text/html"))
}

// q-values decide which representation wins when the client accepts both.
func (s *ContentNegotiationSuite) Test_HonoursQValues() {
	s.mockFile("/docs.md", "# Docs\n")
	s.mockFile("/docs.html", "<h1>Docs</h1>")

	markdownPreferred := s.request(s.host(), "/docs", "text/html;q=0.5, text/markdown;q=0.9")
	s.Equal("text/markdown; charset=utf-8", markdownPreferred.Headers.Get("Content-Type"))

	htmlPreferred := s.request(s.host(), "/docs", "text/html;q=0.9, text/markdown;q=0.5")
	s.True(strings.HasPrefix(htmlPreferred.Headers.Get("Content-Type"), "text/html"))
}

// Explicitly requesting the .md URL serves markdown regardless of Accept.
func (s *ContentNegotiationSuite) Test_DirectMarkdownUrl() {
	s.mockFile("/docs.md", "# Docs\n")

	res := s.request(s.host(), "/docs.md", "text/markdown")

	s.Equal(http.StatusOK, res.Status)
	s.Equal("text/markdown; charset=utf-8", res.Headers.Get("Content-Type"))
}

// A page without a markdown twin is not negotiable, so nothing about its
// response changes — no Vary, no 406.
func (s *ContentNegotiationSuite) Test_PageWithoutTwinIsUnchanged() {
	s.mockFile("/pricing.html", "<h1>Pricing</h1>")

	res := s.request(s.host(), "/pricing", "text/markdown")

	s.Equal(http.StatusOK, res.Status)
	s.True(strings.HasPrefix(res.Headers.Get("Content-Type"), "text/html"))
	s.Empty(res.Headers.Get("Vary"))
}

func (s *ContentNegotiationSuite) Test_NotAcceptable() {
	res := s.request(s.host(), "/docs", "application/pdf")

	s.Equal(http.StatusNotAcceptable, res.Status)
	s.Equal("Accept", res.Headers.Get("Vary"))
}

// The 404 has a markdown representation too, so an agent that hits a dead URL
// can read where to go next.
func (s *ContentNegotiationSuite) Test_MarkdownNotFound() {
	host := s.host()
	host.Config.StaticFiles["/404.html"] = &appconf.StaticFile{FileName: "/404.html"}
	host.Config.StaticFiles["/404.md"] = &appconf.StaticFile{FileName: "/404.md"}

	s.mockFile("/404.md", "# Not found\n\nSee /sitemap.xml\n")

	res := s.request(host, "/no-such-page", "text/markdown")

	s.Equal(http.StatusNotFound, res.Status)
	s.Equal("text/markdown; charset=utf-8", res.Headers.Get("Content-Type"))
	s.Equal("Accept", res.Headers.Get("Vary"))
	s.Contains(string(res.Data.([]byte)), "Not found")
}

func (s *ContentNegotiationSuite) Test_HtmlNotFoundStillWins() {
	host := s.host()
	host.Config.StaticFiles["/404.html"] = &appconf.StaticFile{FileName: "/404.html"}
	host.Config.StaticFiles["/404.md"] = &appconf.StaticFile{FileName: "/404.md"}

	s.mockFile("/404.html", "<h1>Not found</h1>")

	res := s.request(host, "/no-such-page", "text/html")

	s.Equal(http.StatusNotFound, res.Status)
	s.True(strings.HasPrefix(res.Headers.Get("Content-Type"), "text/html"))
	s.Equal("Accept", res.Headers.Get("Vary"))
}

// A static deployment that ships API functions must answer unmatched paths with
// its own error page rather than forwarding them to the API function, which
// would burn an invocation and reply with a bodyless 404.
func (s *ContentNegotiationSuite) Test_UnmatchedPathDoesNotReachApiFunction() {
	host := s.host()
	host.Config.APILocation = "aws:my-api-function"
	host.Config.APIPathPrefix = "/api"
	host.Config.StaticFiles["/404.html"] = &appconf.StaticFile{FileName: "/404.html"}

	s.mockFile("/404.html", "<h1>Not found</h1>")

	res := s.request(host, "/no-such-page", "text/html")

	s.Equal(http.StatusNotFound, res.Status)
	s.Equal("<h1>Not found</h1>", string(res.Data.([]byte)))
	s.mockClient.AssertNotCalled(s.T(), "Invoke")
}

func TestContentNegotiationSuite(t *testing.T) {
	suite.Run(t, new(ContentNegotiationSuite))
}
