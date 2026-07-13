package hosting_test

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"net/http"
	"net/url"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	hosting "github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type SnippetInjectSuite struct {
	suite.Suite
}

func (s *SnippetInjectSuite) request(host *hosting.Host, path string) *hosting.RequestContext {
	rq := &hosting.RequestContext{
		Host: host,
		RequestContext: shttp.NewRequestContext(&http.Request{
			Header: http.Header{},
			URL:    &url.URL{Path: path},
		}),
	}

	rq.OriginalPath = path

	return rq
}

func snippetHost(snippets appconf.Snippets) *hosting.Host {
	return &hosting.Host{
		Name:   "x",
		Config: &appconf.Config{Snippets: snippets},
	}
}

func gzipBytes(s string) []byte {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, _ = w.Write([]byte(s))
	_ = w.Close()

	return buf.Bytes()
}

func deflateBytes(s string) []byte {
	var buf bytes.Buffer
	w, _ := flate.NewWriter(&buf, flate.DefaultCompression)
	_, _ = w.Write([]byte(s))
	_ = w.Close()

	return buf.Bytes()
}

func encodedResponse(encoding string, data []byte) *shttp.Response {
	headers := http.Header{"Content-Type": []string{"text/html"}}

	if encoding != "" {
		headers.Set("Content-Encoding", encoding)
	}

	return &shttp.Response{
		Status:  http.StatusOK,
		Headers: headers,
		Data:    data,
	}
}

const sampleHTML = "<html><head></head><body></body></html>"

func headAppendSnippet() appconf.Snippets {
	return appconf.Snippets{{Content: "<!--sk-head-->", Location: "head"}}
}

func (s *SnippetInjectSuite) Test_InjectsIntoGzipHTML() {
	res := encodedResponse("gzip", gzipBytes(sampleHTML))

	out := hosting.InjectSnippets(s.request(snippetHost(headAppendSnippet()), "/"), res)

	body, ok := out.Data.(string)
	s.Require().True(ok, "body should be decoded to a plaintext string")
	s.Contains(body, "<!--sk-head--></head>")

	// Stale framing/encoding must be dropped so the edge can re-compress.
	s.Empty(out.Headers.Get("Content-Encoding"))
	s.Empty(out.Headers.Get("Content-Length"))
}

func (s *SnippetInjectSuite) Test_InjectsIntoDeflateHTML() {
	res := encodedResponse("deflate", deflateBytes(sampleHTML))

	out := hosting.InjectSnippets(s.request(snippetHost(headAppendSnippet()), "/"), res)

	body, ok := out.Data.(string)
	s.Require().True(ok)
	s.Contains(body, "<!--sk-head--></head>")
	s.Empty(out.Headers.Get("Content-Encoding"))
}

func (s *SnippetInjectSuite) Test_PlainHTMLStripsStaleContentLength() {
	res := encodedResponse("", []byte(sampleHTML))
	res.Headers.Set("Content-Length", "40") // original length, stale after injection

	out := hosting.InjectSnippets(s.request(snippetHost(headAppendSnippet()), "/"), res)

	body, ok := out.Data.(string)
	s.Require().True(ok)
	s.Contains(body, "<!--sk-head--></head>")
	s.Empty(out.Headers.Get("Content-Length"))
}

func (s *SnippetInjectSuite) Test_SkipsUndecodableEncoding() {
	original := []byte("this-is-brotli-or-something")
	res := encodedResponse("br", original)

	out := hosting.InjectSnippets(s.request(snippetHost(headAppendSnippet()), "/"), res)

	// Undecodable encoding: response left completely untouched.
	s.Equal("br", out.Headers.Get("Content-Encoding"))
	data, ok := out.Data.([]byte)
	s.Require().True(ok)
	s.Equal(original, data)
}

func (s *SnippetInjectSuite) Test_NoMatchingTagsKeepsEncoding() {
	// Valid gzip HTML but without <head>/<body> tags to inject around.
	res := encodedResponse("gzip", gzipBytes("<html>no tags here</html>"))

	out := hosting.InjectSnippets(s.request(snippetHost(headAppendSnippet()), "/"), res)

	// Nothing matched, so the original (still-gzipped) response is preserved.
	s.Equal("gzip", out.Headers.Get("Content-Encoding"))
	_, ok := out.Data.([]byte)
	s.True(ok, "body should remain the original encoded bytes")
}

func (s *SnippetInjectSuite) Test_SkipsNonHTML() {
	res := &shttp.Response{
		Status:  http.StatusOK,
		Headers: http.Header{"Content-Type": []string{"application/json"}},
		Data:    `{"ok":true}`,
	}

	out := hosting.InjectSnippets(s.request(snippetHost(headAppendSnippet()), "/"), res)

	s.Equal(`{"ok":true}`, out.Data)
}

func TestSnippetInjectSuite(t *testing.T) {
	suite.Run(t, &SnippetInjectSuite{})
}
