package hosting_test

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/pool"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type HandlerForwardSuite struct {
	suite.Suite

	mockClient  *mocks.ClientInterface
	mockRequest *mocks.RequestInterface
	host        *hosting.Host
	tmpDir      string
}

func (s *HandlerForwardSuite) SetupSuite() {
	s.mockRequest = &mocks.RequestInterface{}

	tmpDir, err := os.MkdirTemp("", "tmp-test-handler-forward-")

	if err != nil {
		panic(err)
	}

	s.tmpDir = tmpDir
}

func (s *HandlerForwardSuite) BeforeTest(_, _ string) {
	rds := rediscache.Client()
	rds.Del(context.Background(), s.mockImageKey())
	rds.Del(context.Background(), "1-/image.jpg") // This is the max image variant

	hosting.Batcher = pool.New(
		pool.WithSize(1000),
		pool.WithFlushInterval(time.Hour),
		pool.WithFlusher(pool.FlusherFunc(func(items []any) {})),
	)

	s.mockClient = &mocks.ClientInterface{}

	integrations.SetDefaultClient(s.mockClient)

	utils.NewUnix = factory.MockNewUnix
	shttp.DefaultRequest = s.mockRequest

	s.host = &hosting.Host{
		Name: "www.stormkit.io",
		Config: &appconf.Config{
			StorageLocation: "aws:my-bucket/my-key-prefix",
			StaticFiles: appconf.StaticFileConfig{
				"/static/index.js": {FileName: "/static/index.js"},
			},
			Redirects: []deploy.Redirect{
				{From: "/_nuxt/*", To: "/*", Assets: true},
				{From: "stormkit.io", To: "www.stormkit.io", Status: 301},
				{From: "staging.stormkit.io", To: "www.stormkit.io/*", Status: 301},
				{From: "/docs/configuration/*", To: "/docs/configuration", Status: 300},
				{From: "/old-blog/*", To: "/new-blog/*", Status: 300},
				{From: "/invalid-(url/*", To: "/invalid", Status: 300},
				{From: "/match", To: "", Status: 300},
				{From: "/test", To: "/static/index.js", Assets: true},
				{From: "/*/metrics/*/metric", To: "/$1/charts/$2/chart", Status: 302},
				{From: "/*/metrics", To: "/$1/charts", Status: 302},
				{From: "/*/metrics/?*", To: "/$1/charts/$2", Status: 302},
				{From: "/api/v1/*", To: "https://test-api.example.com/api/v1/$1"},
				{From: "/api/v2/*", To: "https://test-api.example.com/api/v2/$1", Status: 200},
			},
		},
	}

	// Required for snippet injections and analytics
	admin.SetMockLicense()
}

func (s *HandlerForwardSuite) AfterTest(_, _ string) {
	admin.ResetMockLicense()
	hosting.QueueName = jobs.HostingQueueName
}

func (s *HandlerForwardSuite) TearDownSuite() {
	if strings.Contains(s.tmpDir, os.TempDir()) {
		os.RemoveAll(s.tmpDir)
	}

	utils.NewUnix = factory.OriginalNewUnix
	shttp.DefaultRequest = nil
	integrations.SetDefaultClient(nil)
}

func (s *HandlerForwardSuite) newRequest(host *hosting.Host, path string, headers ...http.Header) *hosting.RequestContext {
	var h http.Header

	if len(headers) > 0 {
		h = headers[0]
	} else {
		h = make(http.Header)
	}

	pieces := strings.Split(path, "?")
	path = pieces[0]
	query := ""

	if len(pieces) > 1 {
		query = pieces[1]
	}

	rq := &hosting.RequestContext{
		Host: host,
		RequestContext: shttp.NewRequestContext(&http.Request{
			Header: h,
			URL: &url.URL{
				Host:     host.Name,
				Path:     path,
				RawQuery: query,
				RawPath:  strings.Split(strings.Split(path, "?")[0], "#")[0],
			},
		}),
	}

	rq.OriginalPath = path

	return rq
}

func (s *HandlerForwardSuite) Test_InjectingHeaders_XRobotsTag() {
	s.mockClient.On("GetFile", integrations.GetFileArgs{
		Location:     "aws:my-bucket/my-key-prefix",
		FileName:     "/some/url/index.html",
		DeploymentID: types.ID(1),
	}).Return(&integrations.GetFileResult{
		Content: []byte("Hello world"),
	}, nil)

	host := &hosting.Host{
		Name: "www.stormkit.io",
		Config: &appconf.Config{
			DeploymentID:    types.ID(1),
			StorageLocation: "aws:my-bucket/my-key-prefix",
			StaticFiles: appconf.StaticFileConfig{
				"/some/url/index.html": {FileName: "/some/url/index.html"},
			},
		},
		IsStormkitSubdomain: true,
	}

	req := s.newRequest(host, "/some/url")
	res := hosting.HandlerForward(req)

	s.Equal(http.StatusOK, res.Status)
	s.Equal("1", res.Headers.Get("x-sk-version"))
	s.Equal("noindex", res.Headers.Get("x-robots-tag"))

	// Now try with a pre-existing x-robots-tag header
	host.Config.StaticFiles["/some/url/index.html"].Headers = map[string]string{
		"x-robots-tag": "index, follow",
	}

	req = s.newRequest(host, "/some/url")
	res = hosting.HandlerForward(req)

	s.Equal(http.StatusOK, res.Status)
	s.Equal("1", res.Headers.Get("x-sk-version"))
	s.Equal("index, follow", res.Headers.Get("x-robots-tag"))

	// Now try with a non stormkit subdomain
	host.IsStormkitSubdomain = false
	host.Config.StaticFiles["/some/url/index.html"].Headers = nil

	req = s.newRequest(host, "/some/url")
	res = hosting.HandlerForward(req)

	s.Equal(http.StatusOK, res.Status)
	s.Equal("1", res.Headers.Get("x-sk-version"))
	s.Equal("", res.Headers.Get("x-robots-tag"))
}

func (s *HandlerForwardSuite) Test_ServeStatic_Success() {
	host := &hosting.Host{
		Name: "www.stormkit.io",
		Request: &shttp.RequestContext{
			Request: &http.Request{},
		},
		Config: &appconf.Config{
			DeploymentID:    types.ID(1),
			EnvID:           types.ID(1),
			StorageLocation: "local:/deployments/deployment-1",
			StaticFiles: appconf.StaticFileConfig{
				"/blog/index.html": &appconf.StaticFile{
					FileName: "/blog/index.html",
					Headers: map[string]string{
						"X-Message":    "Hello-World",
						"content-type": "text/html; charset=utf-8",
					},
				},
			},
		},
	}

	req := s.newRequest(host, "/blog")

	s.mockClient.On("GetFile", integrations.GetFileArgs{
		Location:     "local:/deployments/deployment-1",
		FileName:     "/blog/index.html",
		DeploymentID: types.ID(1),
	}).Return(&integrations.GetFileResult{
		Content: []byte("Hello world"),
	}, nil)

	res := hosting.HandlerForward(req)

	s.Equal(http.StatusOK, res.Status)
	s.Equal([]byte("Hello world"), res.Data)
	s.Equal("text/html; charset=utf-8", res.Headers.Get("content-type"))
	s.Equal("Hello-World", res.Headers.Get("x-message"))
}

func (s *HandlerForwardSuite) Test_ServeDynamic_ServerCmd() {
	host := &hosting.Host{
		Name: "www.stormkit.io",
		Request: &shttp.RequestContext{
			Request: &http.Request{},
		},
		Config: &appconf.Config{
			DeploymentID:     types.ID(1),
			EnvID:            types.ID(1),
			AppID:            types.ID(2),
			FunctionLocation: "local:my-function/10",
			ServerCmd:        "node index.js",
			APIPathPrefix:    "/my/prefix",
		},
	}

	req := s.newRequest(host, "/some/url")

	returnHeaders := make(http.Header)
	returnHeaders.Add("content-type", "text")

	s.mockClient.On("Invoke", mock.MatchedBy(func(args integrations.InvokeArgs) bool {
		s.Equal("www.stormkit.io", args.HostName)
		s.Equal("local:my-function/10", args.ARN)
		s.Equal("node index.js", args.Command)
		s.Equal("/some/url", args.URL.Path)
		s.Equal("/my/prefix", args.Context["apiPrefix"])
		s.Equal(types.ID(2), args.AppID)
		s.Equal(types.ID(1), args.EnvID)
		s.Equal(types.ID(1), args.DeploymentID)
		s.NotNil(args.QueueLog)
		s.True(args.CaptureLogs)
		return true
	})).Return(&integrations.InvokeResult{
		Headers:    returnHeaders,
		StatusCode: http.StatusCreated,
		Body:       []byte(`Hello World`),
	}, nil)

	res := hosting.HandlerForward(req)

	s.Equal(http.StatusCreated, res.Status)
	s.Equal([]byte("Hello World"), res.Data)
	s.Equal("text", res.Headers.Get("content-type"))
	s.Equal("1", res.Headers.Get("x-sk-version"))
}

func (s *HandlerForwardSuite) Test_Redirects_Rewrite() {
	s.mockClient.On("GetFile", integrations.GetFileArgs{
		Location: "aws:my-bucket/my-key-prefix",
		FileName: "/static/index.js",
	}).Return(&integrations.GetFileResult{
		Content: []byte("Hello world"),
	}, nil)

	req := s.newRequest(s.host, "/_nuxt/static/index.js")
	res := hosting.HandlerForward(req)

	s.Nil(res.Redirect)
	s.Equal("/static/index.js", req.URL().Path)
	s.Equal(http.StatusOK, res.Status)
}

func (s *HandlerForwardSuite) Test_Redirects_Rewrite_WithParams() {
	s.mockClient.On("GetFile", integrations.GetFileArgs{
		Location: "aws:my-bucket/my-key-prefix",
		FileName: "/static/index.js",
	}).Return(&integrations.GetFileResult{
		Content: []byte("Hello world"),
	}, nil)

	req := s.newRequest(s.host, "/test?name=savas&surname=vedova")
	res := hosting.HandlerForward(req)

	s.Equal("/static/index.js", req.URL().Path)
	s.Equal("name=savas&surname=vedova", req.URL().RawQuery)
	s.Equal(http.StatusOK, res.Status)
}

func (s *HandlerForwardSuite) Test_Redirects_MultipleEndpointsToSingleEndpoint() {
	req := s.newRequest(s.host, "/docs/configuration/deployments/nuxt")
	res := hosting.HandlerForward(req)

	s.Equal("http://www.stormkit.io/docs/configuration", *res.Redirect)
	s.Equal(300, res.Status)
}

func (s *HandlerForwardSuite) Test_Redirects_RewriteAndRedirectWithQueryParams() {
	req := s.newRequest(s.host, "/old-blog/post-1?sk=1")
	res := hosting.HandlerForward(req)

	s.Equal("http://www.stormkit.io/new-blog/post-1?sk=1", *res.Redirect)
	s.Equal(300, res.Status)
}

func (s *HandlerForwardSuite) Test_Redirects_DomainRewrite() {
	s.host.Name = "stormkit.io"
	req := s.newRequest(s.host, "/some-url/with?query=string")
	res := hosting.HandlerForward(req)

	s.Equal("http://www.stormkit.io/some-url/with?query=string", *res.Redirect)
	s.Equal(301, res.Status)

	s.host.Name = "staging.stormkit.io"
	req = s.newRequest(s.host, "/some-url/with?query=string")
	res = hosting.HandlerForward(req)

	s.Equal("http://www.stormkit.io/some-url/with?query=string", *res.Redirect)
	s.Equal(301, res.Status)
}

func (s *HandlerForwardSuite) Test_Redirects_NoMatchURL() {
	req := s.newRequest(s.host, "/docs/config/deployments")
	res := hosting.HandlerForward(req)

	s.Equal("/docs/config/deployments", req.URL().Path)
	s.Equal(http.StatusNotFound, res.Status)
}

func (s *HandlerForwardSuite) Test_Redirects_MatchWithNoToStatement() {
	// Testing match with no `To` statement
	req := s.newRequest(s.host, "/match")
	res := hosting.HandlerForward(req)

	s.Equal("/match", req.URL().Path)
	s.Equal(http.StatusNotFound, res.Status)
}

// func (s *HandlerForwardSuite) Test_Redirects_APIUrl_WithAPILocation() {
// 	s.host.Config.APILocation = "aws:function:location/41"
// 	s.host.Config.APIPathPrefix = "/api"
// 	s.host.Config.Redirects = []deploy.Redirect{{
// 		From: "/*", To: "/index.html", Status: 300,
// 	}}

// 	s.mockServerlessFn.On("Invoke", mock.MatchedBy(func(_args integrations.AWSInvokeArgs) bool {
// 		return _args.FunctionName == "function:location" && _args.FunctionVersion == "41"
// 	})).Return(&integrations.InvokeResult{
// 		Payload: []byte(`{"statusCode":201,"body":"Hello World","headers":{"content-type":"text"}}`),
// 	}, nil)

// 	req := s.newRequest(s.host, "/api/user/delete")
// 	res := hosting.HandlerForward(req)

// 	s.Nil(res.Redirect)

// 	s.Equal("/api/user/delete", req.URL().Path)
// 	s.Equal(http.StatusCreated, res.Status)
// 	s.host.Config.APILocation = ""
// }

func (s *HandlerForwardSuite) Test_Redirects_APIUrl_WithoutAPILocation() {
	s.host.Config.Redirects = []deploy.Redirect{{
		From: "/*", To: "/index.html", Status: 300,
	}}

	req := s.newRequest(s.host, "/api/user/delete")
	res := hosting.HandlerForward(req)

	s.Equal("http://www.stormkit.io/index.html", *res.Redirect)
	s.Equal(300, res.Status)
}

func (s *HandlerForwardSuite) Test_Redirects_RegexpToPattern() {
	// Test regexp `to` pattern
	req := s.newRequest(s.host, "/stormkitio/metrics")
	res := hosting.HandlerForward(req)

	s.Equal("http://www.stormkit.io/stormkitio/charts", *res.Redirect)
	s.Equal(302, res.Status)

	req = s.newRequest(s.host, "/stormkitio/metrics/4391919/metric")
	res = hosting.HandlerForward(req)

	s.Equal("http://www.stormkit.io/stormkitio/charts/4391919/chart", *res.Redirect)
	s.Equal(302, res.Status)

	req = s.newRequest(s.host, "/stormkitio/metrics/4391919")
	res = hosting.HandlerForward(req)

	s.Equal("http://www.stormkit.io/stormkitio/charts/4391919", *res.Redirect)
	s.Equal(302, res.Status)
}

func (s *HandlerForwardSuite) Test_Redirects_RedirectingToDifferentDomain_ProxyWithStatus() {
	req := s.newRequest(s.host, "/api/v2/my-endpoint")
	req.Body = io.NopCloser(strings.NewReader("my-payload"))

	s.mockRequest.On("URL", "https://test-api.example.com/api/v2/my-endpoint").Return(s.mockRequest).Once()
	s.mockRequest.On("Method", "").Return(s.mockRequest).Once()
	s.mockRequest.On("Headers", shttp.HeadersFromMap(map[string]string{})).Return(s.mockRequest).Once()
	s.mockRequest.On("Payload", req.Body).Return(s.mockRequest).Once()
	s.mockRequest.On("Do").Return(&shttp.HTTPResponse{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("my-response")),
			Header:     make(http.Header),
		},
	}, nil).Once()

	res := hosting.HandlerForward(req)
	data, ok := res.Data.([]byte)

	s.Nil(res.Redirect)
	s.True(ok)
	s.Equal([]byte("my-response"), data)
	s.Equal(http.StatusOK, res.Status)
}

func (s *HandlerForwardSuite) Test_Redirects_RedirectingToDifferentDomain_ProxyWithoutStatus() {
	req := s.newRequest(s.host, "/api/v1/my-endpoint/")
	req.Body = io.NopCloser(strings.NewReader("my-payload"))

	s.mockRequest.On("URL", "https://test-api.example.com/api/v1/my-endpoint/").Return(s.mockRequest).Once()
	s.mockRequest.On("Method", "").Return(s.mockRequest).Once()
	s.mockRequest.On("Headers", shttp.HeadersFromMap(map[string]string{})).Return(s.mockRequest).Once()
	s.mockRequest.On("Payload", req.Body).Return(s.mockRequest).Once()
	s.mockRequest.On("Do").Return(&shttp.HTTPResponse{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("my-response")),
			Header:     make(http.Header),
		},
	}, nil).Once()

	res := hosting.HandlerForward(req)
	data, ok := res.Data.([]byte)

	s.Nil(res.Redirect)
	s.True(ok)
	s.Equal([]byte("my-response"), data)
	s.Equal(http.StatusOK, res.Status)
}

func (s *HandlerForwardSuite) Test_Redirects_UI_Defined_Redirects() {
	req := s.newRequest(s.host, "/docs/configuration/deployments/nuxt")
	req.Host.Config.Redirects = []redirects.Redirect{
		{
			From:   "/docs/configuration/deployments/nuxt",
			To:     "/overwrite",
			Status: 300,
		},
	}

	res := hosting.HandlerForward(req)

	s.Equal("http://www.stormkit.io/overwrite", *res.Redirect)
	s.Equal(300, res.Status)
}

func (s *HandlerForwardSuite) Test_Analytics() {
	s.mockClient.On("GetFile", integrations.GetFileArgs{
		Location:     "aws:my-bucket/my-key-prefix",
		FileName:     "/analytics/index.html",
		DeploymentID: types.ID(1),
	}).Return(&integrations.GetFileResult{
		Content: []byte("Hello world"),
	}, nil)

	host := &hosting.Host{
		Name: "www.stormkit.io",
		Config: &appconf.Config{
			IsEnterprise:    true,
			DeploymentID:    types.ID(1),
			AppID:           types.ID(25),
			EnvID:           types.ID(100),
			DomainID:        types.ID(501),
			StorageLocation: "aws:my-bucket/my-key-prefix",
			StaticFiles: appconf.StaticFileConfig{
				"/analytics/index.html": {
					FileName: "/analytics/index.html",
					Headers: map[string]string{
						"content-type": "text/html; charset=utf-8",
					},
				},
			},
		},
	}

	req := s.newRequest(host, "/analytics?w=1")
	req.Header.Add("X-Forwarded-For", "1.24.15.16")
	req.Header.Add("User-Agent", "mozilla test agent")
	res := hosting.HandlerForward(req)

	s.Equal(http.StatusOK, res.Status)

	s.Eventually(func() bool {
		item := hosting.Batcher.Items(0)

		if item == nil {
			return false
		}

		s.Equal(&jobs.HostingRecord{
			AppID:        types.ID(25),
			EnvID:        types.ID(100),
			DeploymentID: types.ID(1),
			HostName:     "www.stormkit.io",
			Analytics: &analytics.Record{
				AppID:       types.ID(25),
				EnvID:       types.ID(100),
				RequestTS:   utils.NewUnix(),
				RequestPath: "/analytics",
				VisitorIP:   "1.24.15.16",
				StatusCode:  http.StatusOK,
				DomainID:    types.ID(501),
				UserAgent:   null.StringFrom("mozilla test agent"),
			},
			TotalBandwidth: 112,
		}, item)

		return true
	}, time.Second*5, time.Millisecond*100)
}

func (s *HandlerForwardSuite) Test_CacheControl_LastModified() {
	updatedAt := utils.NewUnix()
	updatedAt.Time = time.Unix(1700489144, 0).UTC()

	host := &hosting.Host{
		Name: "www.stormkit.io",
		Config: &appconf.Config{
			DeploymentID:    types.ID(1),
			AppID:           types.ID(25),
			EnvID:           types.ID(100),
			StorageLocation: "aws:my-bucket/my-key-prefix",
			UpdatedAt:       updatedAt,
			StaticFiles: appconf.StaticFileConfig{
				"/some/url/index.html": {
					FileName: "/some/url/index.html",
					Headers: map[string]string{
						"content-type": "text/html; charset=utf-8",
						"etag":         "123",
					},
				},
			},
		},
	}

	req := s.newRequest(host, "/some/url?w=1")
	req.Header.Add("If-Modified-Since", "Sat, 19 Dec 2023 11:25:44 GMT")

	res := hosting.HandlerForward(req)

	s.Equal(http.StatusNotModified, res.Status)
	s.Equal("no-cache, must-revalidate", res.Headers.Get("Cache-Control"))
	s.Equal("Mon, 20 Nov 2023 14:05:44 GMT", res.Headers.Get("Last-Modified"))
	s.Nil(res.Data)
}

func (s *HandlerForwardSuite) Test_CacheControl_ETag() {
	updatedAt := utils.NewUnix()
	updatedAt.Time = time.Unix(1700489144, 0).UTC()

	host := &hosting.Host{
		Name: "www.stormkit.io",
		Config: &appconf.Config{
			DeploymentID:    types.ID(1),
			AppID:           types.ID(25),
			EnvID:           types.ID(100),
			StorageLocation: "aws:my-bucket/my-key-prefix",
			UpdatedAt:       updatedAt,
			StaticFiles: appconf.StaticFileConfig{
				"/some/url/index.html": {
					FileName: "/some/url/index.html",
					Headers: map[string]string{
						"content-type": "text/html; charset=utf-8",
						"etag":         "123",
					},
				},
			},
		},
	}

	req := s.newRequest(host, "/some/url?w=1")
	req.Header.Add("If-None-Match", "123")

	res := hosting.HandlerForward(req)

	s.Equal(http.StatusNotModified, res.Status)
	s.Equal("no-cache, must-revalidate", res.Headers.Get("Cache-Control"))
	s.Equal("Mon, 20 Nov 2023 14:05:44 GMT", res.Headers.Get("Last-Modified"))
	s.Nil(res.Data)
}

func (s *HandlerForwardSuite) Test_CacheControl_ETag_WithIFModifiedSince() {
	s.mockClient.On("GetFile", integrations.GetFileArgs{
		Location:     "aws:my-bucket/my-key-prefix",
		FileName:     "/some/url/index.html",
		DeploymentID: types.ID(1),
	}).Return(&integrations.GetFileResult{
		Content: []byte("Hello world"),
	}, nil)

	updatedAt := utils.NewUnix()
	updatedAt.Time = time.Unix(1700489144, 0).UTC()

	host := &hosting.Host{
		Name: "www.stormkit.io",
		Config: &appconf.Config{
			DeploymentID:    types.ID(1),
			AppID:           types.ID(25),
			EnvID:           types.ID(100),
			StorageLocation: "aws:my-bucket/my-key-prefix",
			UpdatedAt:       updatedAt,
			StaticFiles: appconf.StaticFileConfig{
				"/some/url/index.html": {
					FileName: "/some/url/index.html",
					Headers: map[string]string{
						"content-type": "text/html; charset=utf-8",
						"etag":         "123",
					},
				},
			},
		},
	}

	req := s.newRequest(host, "/some/url?w=1")
	req.Header.Add("If-None-Match", "123")
	req.Header.Add("If-Modified-Since", "Sat, 19 Dec 2022 11:25:44 GMT")

	res := hosting.HandlerForward(req)

	s.Equal(http.StatusOK, res.Status)
	s.Equal("no-cache, must-revalidate", res.Headers.Get("Cache-Control"))
	s.Equal("Mon, 20 Nov 2023 14:05:44 GMT", res.Headers.Get("Last-Modified"))
	s.NotNil(res.Data)
}

func (s *HandlerForwardSuite) Test_404() {
	s.mockClient.On("GetFile", integrations.GetFileArgs{
		Location:     "aws:my-bucket/my-key-prefix",
		FileName:     "/404.html",
		DeploymentID: types.ID(1),
	}).Return(&integrations.GetFileResult{
		Content: []byte("Not found"),
	}, nil)

	host := &hosting.Host{
		Name: "www.stormkit.io",
		Config: &appconf.Config{
			DeploymentID:    types.ID(1),
			AppID:           types.ID(25),
			EnvID:           types.ID(100),
			StorageLocation: "aws:my-bucket/my-key-prefix",
			StaticFiles: appconf.StaticFileConfig{
				"/404.html": {
					FileName: "/404.html",
				},
			},
		},
	}

	req := s.newRequest(host, "/some/url")
	res := hosting.HandlerForward(req)

	s.Equal(http.StatusNotFound, res.Status)
	s.Equal([]byte("Not found"), res.Data.([]byte))
}

func (s *HandlerForwardSuite) Test_404_CustomErrorFile() {
	s.mockClient.On("GetFile", integrations.GetFileArgs{
		Location:     "aws:my-bucket/my-key-prefix",
		FileName:     "/custom-404.html",
		DeploymentID: types.ID(1),
	}).Return(&integrations.GetFileResult{
		Content: []byte("Not found"),
	}, nil)

	host := &hosting.Host{
		Name: "www.stormkit.io",
		Config: &appconf.Config{
			DeploymentID:    types.ID(1),
			AppID:           types.ID(25),
			EnvID:           types.ID(100),
			StorageLocation: "aws:my-bucket/my-key-prefix",
			ErrorFile:       "/custom-404.html",
			StaticFiles: appconf.StaticFileConfig{
				"/custom-404.html": {FileName: "/custom-404.html"},
			},
		},
	}

	req := s.newRequest(host, "/some/url")
	res := hosting.HandlerForward(req)

	s.Equal(http.StatusNotFound, res.Status)
	s.Equal([]byte("Not found"), res.Data.([]byte))
}

func (s *HandlerForwardSuite) mockImageKey() string {
	return "1:10x10/image.jpg"
}

func (s *HandlerForwardSuite) mockImage() []byte {
	// Create a minimal PNG image (1x1 red pixel)
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255}) // Red pixel

	// Encode to PNG
	var buf bytes.Buffer

	s.NoError(png.Encode(&buf, img))

	// Get the raw bytes
	return buf.Bytes()
}

func (s *HandlerForwardSuite) Test_ImageOptimization() {
	s.mockClient.On("GetFile", integrations.GetFileArgs{
		Location:     "aws:my-bucket/my-key-prefix",
		FileName:     "/image.jpg",
		DeploymentID: types.ID(1),
	}).Return(&integrations.GetFileResult{
		Content: s.mockImage(),
	}, nil)

	host := &hosting.Host{
		Name: "www.stormkit.io",
		Config: &appconf.Config{
			DeploymentID:    types.ID(1),
			AppID:           types.ID(25),
			EnvID:           types.ID(100),
			StorageLocation: "aws:my-bucket/my-key-prefix",
			StaticFiles: appconf.StaticFileConfig{
				"/image.jpg": {
					FileName: "/image.jpg",
					Headers: map[string]string{
						"content-type": "image/jpeg",
					},
				},
			},
		},
	}

	req := s.newRequest(host, "/image.jpg?size=10x10")
	res := hosting.HandlerForward(req)

	// Should be cached
	content, err := rediscache.Client().Get(context.Background(), s.mockImageKey()).Result()
	s.NoError(err)
	s.NotNil(content)

	s.Equal(http.StatusOK, res.Status)
	s.Equal([]byte(content), res.Data.([]byte))
}

func (s *HandlerForwardSuite) Test_ImageOptimization_PreviouslyCached() {
	s.NoError(rediscache.Client().Set(context.Background(), s.mockImageKey(), "Image Content", time.Second*20).Err())

	host := &hosting.Host{
		Name: "www.stormkit.io",
		Config: &appconf.Config{
			DeploymentID:    types.ID(1),
			AppID:           types.ID(25),
			EnvID:           types.ID(100),
			StorageLocation: "aws:my-bucket/my-key-prefix",
			StaticFiles: appconf.StaticFileConfig{
				"/image.jpg": {
					FileName: "/image.jpg",
					Headers: map[string]string{
						"content-type": "image/jpeg",
					},
				},
			},
		},
	}

	req := s.newRequest(host, "/image.jpg?size=10x10")
	req.Header.Add("Accept", "image/webp")

	res := hosting.HandlerForward(req)

	s.Equal(http.StatusOK, res.Status)
	s.Equal([]byte("Image Content"), res.Data.([]byte))
}

func (s *HandlerForwardSuite) Test_AuthWall_LoginPage() {
	admin.MustConfig().SetURL("http://stormkit:8888")

	host := &hosting.Host{
		Name: "www.stormkit.io",
		Config: &appconf.Config{
			AuthWall: "all",
		},
	}

	req := s.newRequest(host, "/my-page?with=query")
	res := hosting.HandlerForward(req)
	data := string(res.Data.([]byte))

	s.Equal(http.StatusOK, res.Status)
	s.Equal("text/html; charset=utf-8", res.Headers.Get("Content-Type"))
	s.Contains(data, `method="POST"`)
	s.Contains(data, `action="http://api.stormkit:8888/auth-wall/login"`)
	s.Contains(data, `<form`)
	s.Contains(data, `</form>`)
	s.Contains(data, `<button class="submit-button" type="submit">Login</button>`)
	s.Contains(data, `<input type="hidden" name="token" value="`)
}

func (s *HandlerForwardSuite) Test_AuthWall_DevDomainOnly() {
	host := &hosting.Host{
		Name: "www.stormkit.io",
		Config: &appconf.Config{
			AuthWall: "dev",
		},
	}

	req := s.newRequest(host, "/my-page?with=query")
	res := hosting.HandlerForward(req)

	s.Equal(http.StatusNotFound, res.Status)
	s.Equal("text/html; charset=utf-8", res.Headers.Get("Content-Type"))
	s.NotContains(res.Data, `method="POST"`)

	// This one should display a login page
	host = &hosting.Host{
		Name: "http://bunny-boe.stormkit:8888",
		Config: &appconf.Config{
			AuthWall: "dev",
		},
		IsStormkitSubdomain: true,
	}

	req = s.newRequest(host, "/my-page?with=query")
	res = hosting.HandlerForward(req)
	data := string(res.Data.([]byte))

	s.Equal(http.StatusOK, res.Status)
	s.Equal("text/html; charset=utf-8", res.Headers.Get("Content-Type"))
	s.Contains(data, `method="POST"`)
}

func (s *HandlerForwardSuite) Test_AuthWall_LoginSuccess() {
	host := &hosting.Host{
		Name: "www.stormkit.io",
		Config: &appconf.Config{
			AuthWall: "all",
		},
	}

	token, err := user.JWT(jwt.MapClaims{})
	s.NoError(err)

	req := s.newRequest(host, fmt.Sprintf("/my-page?a=b&stormkit_success=%s", token))
	res := hosting.HandlerForward(req)

	s.Empty(res.Data)
	s.Equal(http.StatusFound, res.Status)
	s.Equal("", res.Headers.Get("Content-Type"))
	s.Equal("http://www.stormkit.io/my-page?a=b", *res.Redirect)
	s.Equal(http.Cookie{
		Name:     hosting.SESSION_COOKIE_NAME,
		Value:    token,
		Expires:  utils.NewUnix().Add(time.Hour * 24),
		SameSite: http.SameSiteStrictMode,
	}, res.Cookies[0])
}

func (s *HandlerForwardSuite) Test_AuthWall_AlreadyLoggedIn() {
	host := &hosting.Host{
		Name: "www.stormkit.io",
		Config: &appconf.Config{
			AuthWall: "all",
		},
	}

	token, err := user.JWT(jwt.MapClaims{})
	s.NoError(err)

	req := s.newRequest(host, "/my-page?a=b")
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", hosting.SESSION_COOKIE_NAME, token))
	res := hosting.HandlerForward(req)
	data := string(res.Data.([]byte))

	// 404 because the host does not contain any information on the CDN file
	s.Equal(http.StatusNotFound, res.Status)
	s.Equal("text/html; charset=utf-8", res.Headers.Get("Content-Type"))
	s.Contains(data, "Whoops! We've got nothing under this link.")
}

func TestHandlerForward(t *testing.T) {
	suite.Run(t, &HandlerForwardSuite{})
}
