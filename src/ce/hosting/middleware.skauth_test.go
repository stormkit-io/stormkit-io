package hosting_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type WithSKAuthSuite struct {
	suite.Suite
}

func (s *WithSKAuthSuite) newRequest(host *hosting.Host, path string) *hosting.RequestContext {
	pieces := strings.Split(path, "?")
	rawPath := pieces[0]
	query := ""

	if len(pieces) > 1 {
		query = pieces[1]
	}

	rq := &hosting.RequestContext{
		Host: host,
		RequestContext: shttp.NewRequestContext(&http.Request{
			Header: make(http.Header),
			URL: &url.URL{
				Host:     host.Name,
				Path:     rawPath,
				RawQuery: query,
				RawPath:  rawPath,
			},
		}),
	}

	rq.OriginalPath = rawPath

	return rq
}

func (s *WithSKAuthSuite) hostWithSKAuth() *hosting.Host {
	return &hosting.Host{
		Name: "www.stormkit.io",
		Config: &appconf.Config{
			SKAuth: &buildconf.SKAuthConf{
				Secret:     "test-secret",
				SuccessURL: "/",
				TTL:        10,
			},
		},
	}
}

func (s *WithSKAuthSuite) newPostRequest(host *hosting.Host, path string, body any) *hosting.RequestContext {
	var buf bytes.Buffer

	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}

	rq := &hosting.RequestContext{
		Host: host,
		RequestContext: shttp.NewRequestContext(&http.Request{
			Method: http.MethodPost,
			Header: make(http.Header),
			URL: &url.URL{
				Host:    host.Name,
				Path:    path,
				RawPath: path,
			},
			Body: io.NopCloser(&buf),
		}),
	}

	rq.OriginalPath = path

	return rq
}


func (s *WithSKAuthSuite) Test_SKAuthDisabled() {
	host := &hosting.Host{
		Name:   "www.stormkit.io",
		Config: &appconf.Config{},
	}

	req := s.newRequest(host, "/_stormkit/auth?code=whatever")
	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.Nil(res)
}

// Test_NonAuthPath checks that the middleware is a no-op for paths that don't
// start with /_stormkit/auth.
func (s *WithSKAuthSuite) Test_NonAuthPath() {
	req := s.newRequest(s.hostWithSKAuth(), "/some/other/path")
	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.Nil(res)
}

// Test_MissingCode checks that the middleware returns 400 when the code query
// parameter is absent.
// Note: the "code is missing" message is overwritten by the Redis lookup path
// ("invalid session") because both branches share the same content variable and
// an empty-key Redis lookup always returns an empty string.
func (s *WithSKAuthSuite) Test_MissingCode() {
	req := s.newRequest(s.hostWithSKAuth(), "/_stormkit/auth")
	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.NotNil(res)
	s.Equal(http.StatusBadRequest, res.Status)
	s.Equal("text/html", res.Headers.Get("Content-Type"))
	s.Contains(string(res.Data.([]byte)), "code is missing")
}

// Test_InvalidCode checks that the middleware returns 200 with an "invalid session"
// message when the submitted code is not found in Redis.
func (s *WithSKAuthSuite) Test_InvalidCode() {
	req := s.newRequest(s.hostWithSKAuth(), "/_stormkit/auth?code=unknown-code")
	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.NotNil(res)
	s.Equal(http.StatusOK, res.Status)
	s.Equal("text/html", res.Headers.Get("Content-Type"))
	s.Contains(string(res.Data.([]byte)), "invalid session")
}

// Test_ValidCode checks that the middleware returns 200 and injects a script that
// stores the session token in localStorage when a valid code is presented.
func (s *WithSKAuthSuite) Test_ValidCode() {
	ctx := context.Background()
	code := "test-valid-code-123"
	sessionToken := "test.session.jwt"

	rds := rediscache.Client()
	rds.Set(ctx, code, sessionToken, time.Minute*2)
	defer rds.Del(ctx, code)

	req := s.newRequest(s.hostWithSKAuth(), "/_stormkit/auth?code="+code)
	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.NotNil(res)
	s.Equal(http.StatusOK, res.Status)
	s.Equal("text/html", res.Headers.Get("Content-Type"))

	body := string(res.Data.([]byte))
	s.Contains(body, `localStorage.setItem('skauth'`)
	s.Contains(body, sessionToken)
}

// Test_RegisterPath_SKAuthDisabled checks that /_stormkit/auth/register is a no-op when SKAuth is not configured.
func (s *WithSKAuthSuite) Test_RegisterPath_SKAuthDisabled() {
	host := &hosting.Host{
		Name:   "www.stormkit.io",
		Config: &appconf.Config{},
	}

	req := s.newPostRequest(host, "/_stormkit/auth/register", map[string]any{
		"email":    "test@example.com",
		"password": "supersecret123",
	})
	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.Nil(res)
}

// Test_RegisterPath_MethodNotAllowed checks that /_stormkit/auth/register rejects non-POST requests.
func (s *WithSKAuthSuite) Test_RegisterPath_MethodNotAllowed() {
	req := s.newRequest(s.hostWithSKAuth(), "/_stormkit/auth/register")
	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.NotNil(res)
	s.Equal(http.StatusMethodNotAllowed, res.Status)
}

// Test_RegisterPath_MissingEnv checks that /_stormkit/auth/register returns 400 when the
// host has no EnvID configured (envId injected as 0 triggers validation failure).
func (s *WithSKAuthSuite) Test_RegisterPath_MissingEnv() {
	req := s.newPostRequest(s.hostWithSKAuth(), "/_stormkit/auth/register", map[string]any{
		"email":    "test@example.com",
		"password": "supersecret123",
	})
	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.NotNil(res)
	s.Equal(http.StatusBadRequest, res.Status)
}

func TestWithSKAuth(t *testing.T) {
	suite.Run(t, new(WithSKAuthSuite))
}
