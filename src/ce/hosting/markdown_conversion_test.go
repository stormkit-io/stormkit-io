package hosting_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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

// page is a document with enough prose to be worth converting.
const page = `<html><head><title>Docs</title><style>b{color:red}</style></head><body>
	<nav><ul><li><a href="/pricing">Pricing</a></li></ul></nav>
	<h1>Deploying</h1>
	<p>Stormkit builds your application and serves the output from the edge, which is enough prose to convert.</p>
	<pre><code class="language-bash">stormkit deploy</code></pre>
	<script>console.log("tracking")</script>
</body></html>`

// shell is what a client-rendered application publishes: markup with no content
// in it until JavaScript runs.
const shell = `<html><body><div id="root"></div><script src="/app.js"></script></body></html>`

// MarkdownConversionSuite covers serving a markdown representation converted
// from a page's own HTML, for deployments that ship no .md twin.
type MarkdownConversionSuite struct {
	suite.Suite

	mockClient *mocks.ClientInterface
}

func (s *MarkdownConversionSuite) BeforeTest(_, _ string) {
	// Conversions are cached by deployment, and every test in this suite uses
	// the same deployment ID, so the cache has to go between them.
	for _, name := range []string{"/docs.html", "/shell.html", "/pricing.html", "/gzipped.html"} {
		rediscache.Client().Del(context.Background(), fmt.Sprintf("md:%s:%s", types.ID(1).String(), name))
	}

	hosting.WaitArtifacts()
	hosting.ResetBatcher(pool.New(
		pool.WithSize(1000),
		pool.WithFlushInterval(time.Hour),
		pool.WithFlusher(pool.FlusherFunc(func(items []any) {})),
	))

	s.mockClient = &mocks.ClientInterface{}
	integrations.SetDefaultClient(s.mockClient)
}

func (s *MarkdownConversionSuite) AfterTest(_, _ string) {
	hosting.WaitArtifacts()
}

// host returns a deployment publishing HTML pages and no markdown twins, so
// every markdown answer has to be converted.
func (s *MarkdownConversionSuite) host(convert bool) *hosting.Host {
	return &hosting.Host{
		Name: "www.stormkit.io",
		Config: &appconf.Config{
			DeploymentID:    types.ID(1),
			AppID:           types.ID(25),
			EnvID:           types.ID(100),
			StorageLocation: "aws:my-bucket/my-key-prefix",
			Markdown:        true,
			MarkdownConvert: convert,
			StaticFiles: appconf.StaticFileConfig{
				"/docs.html": {
					FileName: "/docs.html",
					Headers: map[string]string{
						"content-type": "text/html; charset=utf-8",
						"etag":         `"abc123"`,
					},
				},
				"/shell.html": {
					FileName: "/shell.html",
					Headers:  map[string]string{"content-type": "text/html; charset=utf-8"},
				},
				"/gzipped.html": {
					FileName: "/gzipped.html",
					Headers: map[string]string{
						"content-type":     "text/html; charset=utf-8",
						"content-encoding": "gzip",
						"cache-control":    "public, max-age=60",
					},
				},
			},
		},
	}
}

// hostWithTwin publishes both a page and an authored markdown twin.
func (s *MarkdownConversionSuite) hostWithTwin() *hosting.Host {
	host := s.host(true)
	host.Config.StaticFiles["/docs.md"] = &appconf.StaticFile{
		FileName: "/docs.md",
		Headers:  map[string]string{"content-type": "text/markdown; charset=utf-8"},
	}

	return host
}

func (s *MarkdownConversionSuite) mockFile(fileName, content, contentType string) {
	s.mockClient.On("GetFile", integrations.GetFileArgs{
		Location:     "aws:my-bucket/my-key-prefix",
		FileName:     fileName,
		DeploymentID: types.ID(1),
	}).Return(&integrations.GetFileResult{
		Content:     []byte(content),
		ContentType: contentType,
	}, nil)
}

func (s *MarkdownConversionSuite) request(host *hosting.Host, path, accept string) *shttp.Response {
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

func (s *MarkdownConversionSuite) Test_ConvertsPageWhenNoTwinExists() {
	s.mockFile("/docs.html", page, "text/html; charset=utf-8")

	res := s.request(s.host(true), "/docs", "text/markdown")

	s.Equal(http.StatusOK, res.Status)
	s.Equal("text/markdown; charset=utf-8", res.Headers.Get("Content-Type"))
	s.Equal("Accept", res.Headers.Get("Vary"))

	body := string(res.Data.([]byte))

	s.Contains(body, "# Deploying")
	s.Contains(body, "```bash")
	s.Contains(body, "[Pricing](/pricing)")
	s.NotContains(body, "tracking")
}

// An authored twin is the site's own words and always beats a derived one.
func (s *MarkdownConversionSuite) Test_AuthoredTwinWinsOverConversion() {
	s.mockFile("/docs.md", "# Authored\n", "text/markdown; charset=utf-8")

	res := s.request(s.hostWithTwin(), "/docs", "text/markdown")

	s.Equal(http.StatusOK, res.Status)
	s.Equal("# Authored\n", string(res.Data.([]byte)))
	s.mockClient.AssertNotCalled(s.T(), "GetFile", integrations.GetFileArgs{
		Location:     "aws:my-bucket/my-key-prefix",
		FileName:     "/docs.html",
		DeploymentID: types.ID(1),
	})
}

// Conversion is its own setting. Enabling markdown alone must keep serving
// exactly what it served before.
func (s *MarkdownConversionSuite) Test_ConversionDisabledServesHtml() {
	s.mockFile("/docs.html", page, "text/html; charset=utf-8")

	res := s.request(s.host(false), "/docs", "text/markdown")

	s.Equal(http.StatusOK, res.Status)
	s.True(strings.HasPrefix(res.Headers.Get("Content-Type"), "text/html"))
	s.Empty(res.Headers.Get("Vary"))
}

func (s *MarkdownConversionSuite) Test_BrowserStillGetsHtml() {
	s.mockFile("/docs.html", page, "text/html; charset=utf-8")

	host := s.host(true)

	s.request(host, "/docs", "text/markdown")

	res := s.request(host, "/docs", "text/html,application/xhtml+xml,*/*;q=0.8")

	s.Equal(http.StatusOK, res.Status)
	s.True(strings.HasPrefix(res.Headers.Get("Content-Type"), "text/html"))
	s.Equal("Accept", res.Headers.Get("Vary"))
}

// A .md URL names the markdown representation outright, so it does not depend
// on the Accept header and is not a negotiated response.
func (s *MarkdownConversionSuite) Test_DirectMarkdownURLIsServed() {
	s.mockFile("/docs.html", page, "text/html; charset=utf-8")

	res := s.request(s.host(true), "/docs.md", "")

	s.Equal(http.StatusOK, res.Status)
	s.Equal("text/markdown; charset=utf-8", res.Headers.Get("Content-Type"))
	s.Empty(res.Headers.Get("Vary"))
	s.Contains(string(res.Data.([]byte)), "# Deploying")
}

// A .md URL names one representation, so it is answered the same way whatever
// the client accepts, and must not claim to vary.
func (s *MarkdownConversionSuite) Test_AuthoredMarkdownURLIsServedDirectly() {
	s.mockFile("/docs.md", "# Authored\n", "text/markdown; charset=utf-8")

	host := s.hostWithTwin()

	for _, accept := range []string{"", "text/html,application/xhtml+xml,*/*;q=0.8", "text/markdown"} {
		res := s.request(host, "/docs.md", accept)

		s.Equal(http.StatusOK, res.Status)
		s.Equal("# Authored\n", string(res.Data.([]byte)))
		s.Empty(res.Headers.Get("Vary"), "accept=%q", accept)
	}
}

// The page it sits beside is only converted when the build published no .md of
// its own.
func (s *MarkdownConversionSuite) Test_AuthoredMarkdownURLBeatsConversion() {
	s.mockFile("/docs.md", "# Authored\n", "text/markdown; charset=utf-8")

	res := s.request(s.hostWithTwin(), "/docs.md", "text/markdown")

	s.Equal("# Authored\n", string(res.Data.([]byte)))
	s.mockClient.AssertNotCalled(s.T(), "GetFile", integrations.GetFileArgs{
		Location:     "aws:my-bucket/my-key-prefix",
		FileName:     "/docs.html",
		DeploymentID: types.ID(1),
	})
}

func (s *MarkdownConversionSuite) Test_DirectMarkdownURLIsNotFoundWhenConversionDisabled() {
	res := s.request(s.host(false), "/docs.md", "")

	s.Equal(http.StatusNotFound, res.Status)
}

// A client-rendered shell has no content until JavaScript runs, so it converts
// to an empty document. Conversion is opt-in: a deployment that enables it and
// gets nothing back has learned something true about its pages, which is a
// better answer than quietly serving the HTML instead.
func (s *MarkdownConversionSuite) Test_EmptyShellConvertsToNothing() {
	s.mockFile("/shell.html", shell, "text/html; charset=utf-8")

	res := s.request(s.host(true), "/shell", "text/markdown")

	s.Equal(http.StatusOK, res.Status)
	s.Equal("text/markdown; charset=utf-8", res.Headers.Get("Content-Type"))
	s.Empty(string(res.Data.([]byte)))
}

// The second request must not re-read or re-parse the page.
func (s *MarkdownConversionSuite) Test_ConversionIsCached() {
	s.mockFile("/docs.html", page, "text/html; charset=utf-8")

	host := s.host(true)

	first := s.request(host, "/docs", "text/markdown")
	second := s.request(host, "/docs", "text/markdown")

	s.Equal(string(first.Data.([]byte)), string(second.Data.([]byte)))
	s.mockClient.AssertNumberOfCalls(s.T(), "GetFile", 1)
}

// An empty conversion is a result like any other, so it is cached and the page
// is not re-parsed on the next request.
func (s *MarkdownConversionSuite) Test_EmptyConversionIsCached() {
	s.mockFile("/shell.html", shell, "text/html; charset=utf-8")

	host := s.host(true)

	s.request(host, "/shell", "text/markdown")
	s.request(host, "/shell", "text/markdown")

	s.mockClient.AssertNumberOfCalls(s.T(), "GetFile", 1)
}

// The two representations of a URL must not validate against each other: a
// client holding one would be told its copy of the other is current.
func (s *MarkdownConversionSuite) Test_MarkdownCarriesItsOwnETag() {
	s.mockFile("/docs.html", page, "text/html; charset=utf-8")

	markdown := s.request(s.host(true), "/docs", "text/markdown")
	html := s.request(s.host(true), "/docs", "text/html")

	s.NotEmpty(markdown.Headers.Get("ETag"))
	s.NotEqual(html.Headers.Get("ETag"), markdown.Headers.Get("ETag"))
}

func (s *MarkdownConversionSuite) Test_MarkdownSharesThePageCachePolicy() {
	s.mockFile("/docs.html", page, "text/html; charset=utf-8")

	res := s.request(s.host(true), "/docs", "text/markdown")

	s.Equal("no-cache, must-revalidate", res.Headers.Get("Cache-Control"))
}

// A build that publishes pre-compressed HTML marks it Content-Encoding: gzip.
// The converted markdown is not gzipped, so carrying that header over would
// produce a response no client can decode.
func (s *MarkdownConversionSuite) Test_DoesNotCarryContentEncoding() {
	s.mockFile("/gzipped.html", page, "text/html; charset=utf-8")

	res := s.request(s.host(true), "/gzipped", "text/markdown")

	s.Equal(http.StatusOK, res.Status)
	s.Equal("text/markdown; charset=utf-8", res.Headers.Get("Content-Type"))
	s.Empty(res.Headers.Get("Content-Encoding"))
	s.Equal("public, max-age=60", res.Headers.Get("Cache-Control"))
}

// Every HTML page in a converting deployment has a markdown representation, so
// the HTML answer says so on the first request — a cache must never be able to
// store the page unkeyed and then hand it to an agent asking for markdown.
func (s *MarkdownConversionSuite) Test_VaryIsSetBeforeAnythingIsConverted() {
	s.mockFile("/docs.html", page, "text/html; charset=utf-8")

	res := s.request(s.host(true), "/docs", "text/html,application/xhtml+xml,*/*;q=0.8")

	s.Equal(http.StatusOK, res.Status)
	s.Equal("Accept", res.Headers.Get("Vary"))
}

func TestMarkdownConversionSuite(t *testing.T) {
	suite.Run(t, new(MarkdownConversionSuite))
}
