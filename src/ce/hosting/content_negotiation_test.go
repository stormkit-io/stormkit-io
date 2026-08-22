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
	"github.com/stretchr/testify/mock"
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
			Markdown:        true,
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

// Both representations of a URL must expire together. Classifying the markdown
// twin by its own content type would file it under the asset policy and leave a
// corrected page stale for a day.
func (s *ContentNegotiationSuite) Test_MarkdownSharesThePageCachePolicy() {
	s.mockFile("/docs.md", "# Docs\n")

	res := s.request(s.host(), "/docs", "text/markdown")

	s.Equal("no-cache, must-revalidate", res.Headers.Get("Cache-Control"))
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

// Negotiation never refuses a request. A client that accepts neither
// representation gets the HTML it would have got before the page became
// negotiable — uptime probes and JSON-only fetch wrappers keep working.
func (s *ContentNegotiationSuite) Test_UnsupportedAcceptStillServesHtml() {
	s.mockFile("/docs.html", "<h1>Docs</h1>")

	for _, accept := range []string{"application/pdf", "application/json", "text/plain"} {
		res := s.request(s.host(), "/docs", accept)

		s.Equal(http.StatusOK, res.Status, "Accept: %s", accept)
		s.True(strings.HasPrefix(res.Headers.Get("Content-Type"), "text/html"), "Accept: %s", accept)
		s.Equal("Accept", res.Headers.Get("Vary"), "Accept: %s", accept)
	}
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

// Negotiation is opt-in: a build that merely happens to publish .md files keeps
// serving exactly what it served before.
func (s *ContentNegotiationSuite) Test_DisabledByDefault() {
	host := s.host()
	host.Config.Markdown = false

	s.mockFile("/docs.html", "<h1>Docs</h1>")

	res := s.request(host, "/docs", "text/markdown")

	s.Equal(http.StatusOK, res.Status)
	s.True(strings.HasPrefix(res.Headers.Get("Content-Type"), "text/html"))
	s.Empty(res.Headers.Get("Vary"))
}

// The HTML representation is not always a static file. When it is server
// rendered the response still has to carry Vary, or a cache keyed on the URL
// alone hands that HTML to the next client asking for markdown.
func (s *ContentNegotiationSuite) Test_VaryOnServerRenderedVariant() {
	host := s.host()
	host.Config.FunctionLocation = "aws:my-server-function"

	delete(host.Config.StaticFiles, "/docs.html")

	s.mockClient.On("Invoke", mock.Anything).Return(&integrations.InvokeResult{
		StatusCode: http.StatusOK,
		Body:       []byte("<h1>Docs</h1>"),
		Headers:    http.Header{"Content-Type": []string{"text/html; charset=utf-8"}},
	}, nil)

	res := s.request(host, "/docs", "text/html")

	s.Equal(http.StatusOK, res.Status)
	s.Equal("Accept", res.Headers.Get("Vary"))
}

// A deployment shipping only a markdown error page still answers browsers with
// the built-in 404 — which must carry Vary just the same.
func (s *ContentNegotiationSuite) Test_VaryOnBuiltInNotFound() {
	host := s.host()
	host.Config.StaticFiles["/404.md"] = &appconf.StaticFile{FileName: "/404.md"}

	res := s.request(host, "/no-such-page", "text/html")

	s.Equal(http.StatusNotFound, res.Status)
	s.Equal("Accept", res.Headers.Get("Vary"))
}

// A deployment that already varies on something keeps that token.
func (s *ContentNegotiationSuite) Test_VaryPreservesExistingTokens() {
	host := s.host()
	host.Config.StaticFiles["/docs.md"].Headers = map[string]string{
		"content-type": "text/markdown; charset=utf-8",
		"vary":         "Accept-Encoding",
	}

	s.mockFile("/docs.md", "# Docs\n")

	res := s.request(host, "/docs", "text/markdown")

	s.Equal("Accept-Encoding, Accept", res.Headers.Get("Vary"))
}

// A SPA points errorFile at its app shell, whose markdown twin is the homepage.
// That must never become the body of a 404.
func (s *ContentNegotiationSuite) Test_MarkdownNotFoundIgnoresConfiguredErrorFile() {
	host := s.host()
	host.Config.ErrorFile = "/index.html"
	host.Config.StaticFiles["/index.html"] = &appconf.StaticFile{FileName: "/index.html"}
	host.Config.StaticFiles["/index.md"] = &appconf.StaticFile{FileName: "/index.md"}

	s.mockFile("/index.html", "<div id=root></div>")

	res := s.request(host, "/no-such-page", "text/markdown")

	s.Equal(http.StatusNotFound, res.Status)
	s.False(strings.HasPrefix(res.Headers.Get("Content-Type"), "text/markdown"),
		"the homepage markdown must not be served as the 404 body")
}

func (s *ContentNegotiationSuite) Test_MarkdownNotFoundAcceptsFiveHundred() {
	host := s.host()
	host.Config.StaticFiles["/500.md"] = &appconf.StaticFile{FileName: "/500.md"}

	s.mockFile("/500.md", "# Error\n")

	res := s.request(host, "/no-such-page", "text/markdown")

	s.Equal(http.StatusNotFound, res.Status)
	s.Equal("text/markdown; charset=utf-8", res.Headers.Get("Content-Type"))
}

func TestContentNegotiationSuite(t *testing.T) {
	suite.Run(t, new(ContentNegotiationSuite))
}
