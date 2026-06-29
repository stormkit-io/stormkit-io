package hosting_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	hosting "github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type AnalyticsScriptSuite struct {
	suite.Suite
}

func (s *AnalyticsScriptSuite) newRequest(host *hosting.Host, path string) *hosting.RequestContext {
	rq := &hosting.RequestContext{
		Host: host,
		RequestContext: shttp.NewRequestContext(&http.Request{
			Header: http.Header{},
			URL:    &url.URL{Path: path},
		}),
	}

	rq.OriginalPath = path
	rq.RequestID = "req-123"

	return rq
}

func htmlResponse(body string) *shttp.Response {
	return &shttp.Response{
		Status:  http.StatusOK,
		Headers: http.Header{"Content-Type": []string{"text/html"}},
		Data:    body,
	}
}

func (s *AnalyticsScriptSuite) Test_ServesScript() {
	req := s.newRequest(&hosting.Host{Name: "x"}, "/_stormkit/analytics.js")
	res, err := hosting.WithAnalyticsScript(req)

	s.NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusOK, res.Status)
	s.Contains(res.Headers.Get("Content-Type"), "javascript")
	s.NotEmpty(res.Headers.Get("ETag"))

	body, _ := res.Data.([]byte)
	s.Contains(string(body), "window.stormkit")
}

func (s *AnalyticsScriptSuite) Test_NotModifiedOnMatchingETag() {
	req := s.newRequest(&hosting.Host{Name: "x"}, "/_stormkit/analytics.js")
	first, _ := hosting.WithAnalyticsScript(req)

	req2 := s.newRequest(&hosting.Host{Name: "x"}, "/_stormkit/analytics.js")
	req2.Header.Set("If-None-Match", first.Headers.Get("ETag"))
	res, _ := hosting.WithAnalyticsScript(req2)

	s.Equal(http.StatusNotModified, res.Status)
}

func (s *AnalyticsScriptSuite) Test_PassesThroughOtherPaths() {
	req := s.newRequest(&hosting.Host{Name: "x"}, "/index.html")
	res, err := hosting.WithAnalyticsScript(req)

	s.NoError(err)
	s.Nil(res)
}

func (s *AnalyticsScriptSuite) Test_ServesEmbeddedDefaultWhenNoOverride() {
	content, etag := hosting.ResolveAnalyticsScript(admin.InstanceConfig{})

	s.Contains(string(content), "window.stormkit")
	s.Equal(hosting.EmbeddedScriptETag(), etag)
}

func (s *AnalyticsScriptSuite) Test_ServesAdminOverride() {
	cfg := admin.InstanceConfig{
		AnalyticsScript: &admin.AnalyticsScriptConfig{
			Content: "console.log('override');",
			Hash:    "deadbeef",
		},
	}

	content, etag := hosting.ResolveAnalyticsScript(cfg)

	s.Equal("console.log('override');", string(content))
	s.Equal(`"deadbeef"`, etag)
	s.NotEqual(hosting.EmbeddedScriptETag(), etag)
}

func (s *AnalyticsScriptSuite) Test_InjectsInterpolatedSnippet() {
	host := &hosting.Host{
		Name: "x",
		Config: &appconf.Config{
			Snippets: appconf.Snippets{
				{
					Content:     `<script src="/_stormkit/analytics.js" data-sk-rid="{{SK_REQUEST_ID}}" async></script>`,
					Location:    "head",
					Interpolate: true,
				},
			},
		},
	}

	out := hosting.InjectSnippets(
		s.newRequest(host, "/"),
		htmlResponse("<html><head></head><body></body></html>"),
	)

	body, _ := out.Data.(string)
	s.Contains(body, `data-sk-rid="req-123"`)
	s.NotContains(body, "{{SK_REQUEST_ID}}")
}

func TestAnalyticsScriptSuite(t *testing.T) {
	suite.Run(t, &AnalyticsScriptSuite{})
}
