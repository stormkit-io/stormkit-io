package hosting_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
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

func (s *WithSKAuthSuite) newGetRequest(host *hosting.Host, path string) *hosting.RequestContext {
	rq := s.newRequest(host, path)
	rq.RequestContext.Method = http.MethodGet
	return rq
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

// Test_CORSPreflight checks that OPTIONS requests to /_stormkit/auth/* return
// 204 so the browser preflight succeeds. The custom CORS headers are layered
// on later by finalize() in HandlerForward.
func (s *WithSKAuthSuite) Test_CORSPreflight() {
	req := s.newRequest(s.hostWithSKAuth(), "/_stormkit/auth/magic")
	req.RequestContext.Method = http.MethodOptions

	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.NotNil(res)
	s.Equal(http.StatusNoContent, res.Status)
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

// Test_RegisterPath_MissingEnv checks that /_stormkit/auth/register returns 404 when the
// host has no EnvID configured — the env lookup finds nothing and auth is unavailable.
func (s *WithSKAuthSuite) Test_RegisterPath_MissingEnv() {
	req := s.newPostRequest(s.hostWithSKAuth(), "/_stormkit/auth/register", map[string]any{
		"email":    "test@example.com",
		"password": "supersecret123",
	})
	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.NotNil(res)
	s.Equal(http.StatusNotFound, res.Status)
}

// Test_LoginPath_SKAuthDisabled checks that /_stormkit/auth/login is a no-op when SKAuth is not configured.
func (s *WithSKAuthSuite) Test_LoginPath_SKAuthDisabled() {
	host := &hosting.Host{
		Name:   "www.stormkit.io",
		Config: &appconf.Config{},
	}

	req := s.newPostRequest(host, "/_stormkit/auth/login", map[string]any{
		"email":    "test@example.com",
		"password": "supersecret123",
	})
	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.Nil(res)
}

// Test_LoginPath_MethodNotAllowed checks that /_stormkit/auth/login rejects non-POST requests.
func (s *WithSKAuthSuite) Test_LoginPath_MethodNotAllowed() {
	req := s.newRequest(s.hostWithSKAuth(), "/_stormkit/auth/login")
	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.NotNil(res)
	s.Equal(http.StatusMethodNotAllowed, res.Status)
}

// Test_LoginPath_MissingEnv checks that /_stormkit/auth/login returns 404 when the
// host has no EnvID configured — the env lookup finds nothing and auth is unavailable.
func (s *WithSKAuthSuite) Test_LoginPath_MissingEnv() {
	req := s.newPostRequest(s.hostWithSKAuth(), "/_stormkit/auth/login", map[string]any{
		"email":    "test@example.com",
		"password": "supersecret123",
	})
	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.NotNil(res)
	s.Equal(http.StatusNotFound, res.Status)
}

// Test_VerifyPath_MethodNotAllowed checks that /_stormkit/auth/verify rejects non-GET requests.
func (s *WithSKAuthSuite) Test_VerifyPath_MethodNotAllowed() {
	req := s.newPostRequest(s.hostWithSKAuth(), "/_stormkit/auth/verify", nil)
	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.NotNil(res)
	s.Equal(http.StatusMethodNotAllowed, res.Status)
}

// Test_VerifyPath_MissingToken checks that /_stormkit/auth/verify returns 400 when no token is provided.
func (s *WithSKAuthSuite) Test_VerifyPath_MissingToken() {
	req := s.newGetRequest(s.hostWithSKAuth(), "/_stormkit/auth/verify")
	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.NotNil(res)
	s.Equal(http.StatusBadRequest, res.Status)
}

// Test_VerifyPath_MissingEnv checks that /_stormkit/auth/verify returns 404 when the host
// has no EnvID configured — the env lookup finds nothing and auth is unavailable.
func (s *WithSKAuthSuite) Test_VerifyPath_MissingEnv() {
	req := s.newGetRequest(s.hostWithSKAuth(), "/_stormkit/auth/verify?token=some-token")
	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.NotNil(res)
	s.Equal(http.StatusNotFound, res.Status)
}

// Test_MagicPath_MissingEnv checks that GET /_stormkit/auth/magic returns 404 when
// the host has no EnvID configured — the zero-envId guard fires first.
func (s *WithSKAuthSuite) Test_MagicPath_MissingEnv() {
	req := s.newGetRequest(s.hostWithSKAuth(), "/_stormkit/auth/magic?email=user@example.com")
	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.NotNil(res)
	s.Equal(http.StatusNotFound, res.Status)
}

// Test_MagicPath_VerifyInvalidToken checks that a token submitted against a host with no
// EnvID configured returns 404 — the zero-envId guard fires before token validation.
func (s *WithSKAuthSuite) Test_MagicPath_VerifyInvalidToken() {
	req := s.newGetRequest(s.hostWithSKAuth(), "/_stormkit/auth/magic?token=bad-token")
	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.NotNil(res)
	s.Equal(http.StatusNotFound, res.Status)
}

func (s *WithSKAuthSuite) generateBearer(userID types.ID, secret string) string {
	token, err := user.JWT(jwt.MapClaims{"uid": userID}, secret)
	s.Require().NoError(err)
	return fmt.Sprintf("Bearer %s", token)
}

// Test_UserIDStripped checks that X-User-Id is removed when SKAuth is configured
// to prevent clients from spoofing the header.
func (s *WithSKAuthSuite) Test_UserIDStripped() {
	req := s.newRequest(s.hostWithSKAuth(), "/some/path")
	req.Header.Set("X-User-Id", "spoofed-id")

	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.Nil(res)
	s.Empty(req.Header.Get("X-User-Id"))
}

// Test_UserIDNotStrippedWhenSKAuthDisabled checks that X-User-Id is left untouched
// when SKAuth is not configured.
func (s *WithSKAuthSuite) Test_UserIDNotStrippedWhenSKAuthDisabled() {
	host := &hosting.Host{
		Name:   "www.stormkit.io",
		Config: &appconf.Config{},
	}

	req := s.newRequest(host, "/some/path")
	req.Header.Set("X-User-Id", "some-id")

	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.Nil(res)
	s.Equal("some-id", req.Header.Get("X-User-Id"))
}

// Test_UserIDInjected checks that a valid bearer token results in X-User-Id being set.
func (s *WithSKAuthSuite) Test_UserIDInjected() {
	userID := types.ID(42)
	req := s.newRequest(s.hostWithSKAuth(), "/api/data")
	req.Header.Set("Authorization", s.generateBearer(userID, "test-secret"))

	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.Nil(res)
	s.Equal(userID.String(), req.Header.Get("X-User-Id"))
}

// Test_UserIDNotInjectedOnInvalidToken checks that an invalid token leaves X-User-Id unset.
func (s *WithSKAuthSuite) Test_UserIDNotInjectedOnInvalidToken() {
	req := s.newRequest(s.hostWithSKAuth(), "/api/data")
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")

	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.Nil(res)
	s.Empty(req.Header.Get("X-User-Id"))
}

// Test_UserIDNotInjectedOnWrongSecret checks that a token signed with a different secret is rejected.
func (s *WithSKAuthSuite) Test_UserIDNotInjectedOnWrongSecret() {
	req := s.newRequest(s.hostWithSKAuth(), "/api/data")
	req.Header.Set("Authorization", s.generateBearer(types.ID(1), "wrong-secret"))

	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.Nil(res)
	s.Empty(req.Header.Get("X-User-Id"))
}

// Test_UserIDNotInjectedWhenNoAuthHeader checks that requests without an Authorization
// header do not get X-User-Id injected.
func (s *WithSKAuthSuite) Test_UserIDNotInjectedWhenNoAuthHeader() {
	req := s.newRequest(s.hostWithSKAuth(), "/api/data")

	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.Nil(res)
	s.Empty(req.Header.Get("X-User-Id"))
}

// generateBearerIssuedAt signs a JWT with a forged "issued" claim so tests
// can exercise expiry behavior without sleeping.
func (s *WithSKAuthSuite) generateBearerIssuedAt(userID types.ID, secret string, issuedAt time.Time) string {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid":    userID,
		"issued": issuedAt.Unix(),
	})

	signed, err := tok.SignedString([]byte(secret))
	s.Require().NoError(err)
	return fmt.Sprintf("Bearer %s", signed)
}

// Test_Refresh_ValidToken checks that POSTing a still-valid bearer to
// /_stormkit/auth/refresh returns a new JWT carrying the same uid.
func (s *WithSKAuthSuite) Test_Refresh_ValidToken() {
	host := s.hostWithSKAuth() // TTL = 10 min
	req := s.newPostRequest(host, "/_stormkit/auth/refresh", nil)
	req.Header.Set("Authorization", s.generateBearer(types.ID(42), "test-secret"))

	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusOK, res.Status)

	data, ok := res.Data.(map[string]any)
	s.Require().True(ok)

	newToken, ok := data["token"].(string)
	s.Require().True(ok)
	s.NotEmpty(newToken)

	claims := user.ParseJWT(&user.ParseJWTArgs{
		Bearer:  newToken,
		Secret:  "test-secret",
		MaxMins: 10,
	})
	s.Require().NotNil(claims)
	s.Equal("42", claims["uid"])
}

// Test_Refresh_ExpiredToken checks that an expired bearer is rejected with
// 401 — expired means expired, no grace window.
func (s *WithSKAuthSuite) Test_Refresh_ExpiredToken() {
	host := s.hostWithSKAuth() // TTL = 10 min
	req := s.newPostRequest(host, "/_stormkit/auth/refresh", nil)
	req.Header.Set("Authorization", s.generateBearerIssuedAt(types.ID(7), "test-secret", time.Now().Add(-30*time.Minute)))

	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusUnauthorized, res.Status)
}

// Test_Refresh_MissingBearer checks that calls without an Authorization
// header are rejected with 401.
func (s *WithSKAuthSuite) Test_Refresh_MissingBearer() {
	host := s.hostWithSKAuth()
	req := s.newPostRequest(host, "/_stormkit/auth/refresh", nil)

	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusUnauthorized, res.Status)
}

// Test_Refresh_WrongMethod checks that non-POST verbs on /refresh return 405.
func (s *WithSKAuthSuite) Test_Refresh_WrongMethod() {
	host := s.hostWithSKAuth()
	req := s.newGetRequest(host, "/_stormkit/auth/refresh")
	req.Header.Set("Authorization", s.generateBearer(types.ID(1), "test-secret"))

	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusMethodNotAllowed, res.Status)
}

func TestWithSKAuth(t *testing.T) {
	suite.Run(t, new(WithSKAuthSuite))
}
