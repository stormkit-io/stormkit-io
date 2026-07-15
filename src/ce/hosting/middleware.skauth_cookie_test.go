package hosting_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

const cookieSecret = "test-secret-padded-to-32-chars!!"

// SkAuthCookieSuite covers cookie-mode session storage: the edge reading the
// session from a cookie, and the login handlers issuing one.
type SkAuthCookieSuite struct {
	suite.Suite
}

func TestSkAuthCookieSuite(t *testing.T) {
	suite.Run(t, new(SkAuthCookieSuite))
}

func (s *SkAuthCookieSuite) host(cookieMode bool) *hosting.Host {
	conf := &buildconf.SKAuthConf{
		Secret:     cookieSecret,
		SuccessURL: "/",
		Status:     true,
		TTL:        10,
	}

	if cookieMode {
		conf.SessionStorage = buildconf.SessionStorageCookie
		conf.CookieDomain = ".example.com"
	}

	return &hosting.Host{
		Name:   "www.example.com",
		Config: &appconf.Config{SKAuth: conf},
	}
}

func (s *SkAuthCookieSuite) token(uid string) string {
	tok, err := user.JWT(jwt.MapClaims{"uid": uid}, cookieSecret)
	s.Require().NoError(err)

	return tok
}

func (s *SkAuthCookieSuite) request(host *hosting.Host, header http.Header) *hosting.RequestContext {
	if header == nil {
		header = make(http.Header)
	}

	rq := &hosting.RequestContext{
		Host: host,
		RequestContext: shttp.NewRequestContext(&http.Request{
			Method: http.MethodGet,
			Header: header,
			URL:    &url.URL{Host: host.Name, Path: "/dashboard", RawPath: "/dashboard"},
		}),
	}

	rq.OriginalPath = "/dashboard"

	return rq
}

// Test_Edge_InjectsUserFromCookie verifies the edge resolves the session from
// the cookie (no Authorization header) and injects the identity headers — the
// core of what makes cookie mode work on top-level navigations.
func (s *SkAuthCookieSuite) Test_Edge_InjectsUserFromCookie() {
	h := make(http.Header)
	h.Set("Cookie", buildconf.SessionCookieName+"="+s.token("user-42"))

	req := s.request(s.host(true), h)
	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.Nil(res, "a normal path falls through after injecting headers")
	s.Equal("user-42", req.Header.Get("X-User-Id"))
}

// Test_Edge_InjectsUserFromAuthorization guards the localStorage/API path: the
// Authorization bearer must still resolve identity after adding cookie support.
func (s *SkAuthCookieSuite) Test_Edge_InjectsUserFromAuthorization() {
	h := make(http.Header)
	h.Set("Authorization", "Bearer "+s.token("user-7"))

	req := s.request(s.host(false), h)
	_, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.Equal("user-7", req.Header.Get("X-User-Id"))
}

// Test_Edge_NoCredential_NoIdentity verifies no session yields no identity
// header (and any client-supplied one is stripped).
func (s *SkAuthCookieSuite) Test_Edge_NoCredential_NoIdentity() {
	h := make(http.Header)
	h.Set("X-User-Id", "spoofed")

	req := s.request(s.host(true), h)
	_, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.Empty(req.Header.Get("X-User-Id"))
}

// Test_Login_CookieMode_SetsCookie verifies the one-time-code landing issues a
// shared, hardened session cookie and 302s — instead of a localStorage script —
// when the environment is in cookie mode.
func (s *SkAuthCookieSuite) Test_Login_CookieMode_SetsCookie() {
	ctx := context.Background()
	code := "cookie-mode-code-1"
	token := s.token("user-99")

	rds := rediscache.Client()
	rds.Set(ctx, code, `{"token":"`+token+`","redirect":"https://app.example.com/dashboard"}`, time.Minute*2)
	defer rds.Del(ctx, code)

	req := s.request(s.host(true), nil)
	req.RequestContext.Method = http.MethodGet
	req.RequestContext.Request.URL = &url.URL{Host: "www.example.com", Path: "/_stormkit/auth", RawQuery: "code=" + code}
	req.OriginalPath = "/_stormkit/auth"

	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusFound, res.Status)
	s.Require().NotNil(res.Redirect)
	s.Equal("https://app.example.com/dashboard", *res.Redirect)

	s.Require().Len(res.Cookies, 1)
	c := res.Cookies[0]
	s.Equal(buildconf.SessionCookieName, c.Name)
	s.Equal(token, c.Value)
	s.Equal(".example.com", c.Domain)
	s.True(c.Secure)
	s.True(c.HttpOnly)
	s.Equal(http.SameSiteLaxMode, c.SameSite)
}

// Test_Login_LocalStorageMode_UsesScript keeps the default behavior: no cookie,
// a localStorage-injecting landing page.
func (s *SkAuthCookieSuite) Test_Login_LocalStorageMode_UsesScript() {
	ctx := context.Background()
	code := "ls-mode-code-1"
	token := "ls.session.jwt"

	rds := rediscache.Client()
	rds.Set(ctx, code, `{"token":"`+token+`","redirect":"https://app.example.com/dashboard"}`, time.Minute*2)
	defer rds.Del(ctx, code)

	req := s.request(s.host(false), nil)
	req.RequestContext.Request.URL = &url.URL{Host: "www.example.com", Path: "/_stormkit/auth", RawQuery: "code=" + code}
	req.OriginalPath = "/_stormkit/auth"

	res, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusOK, res.Status)
	s.Empty(res.Cookies)
	s.Contains(string(res.Data.([]byte)), "localStorage.setItem('skauth'")
}

// logoutRequest builds a request to the logout endpoint with the given method.
func (s *SkAuthCookieSuite) logoutRequest(host *hosting.Host, method string) *hosting.RequestContext {
	req := s.request(host, nil)
	req.RequestContext.Method = method
	req.RequestContext.Request.URL = &url.URL{Host: host.Name, Path: "/_stormkit/auth/logout", RawPath: "/_stormkit/auth/logout"}
	req.OriginalPath = "/_stormkit/auth/logout"

	return req
}

// Test_Logout_ExpiresCookie verifies POST /_stormkit/auth/logout clears the
// session cookie — the only way to end a cookie-mode session, since the HttpOnly
// cookie is invisible to the app's JS.
func (s *SkAuthCookieSuite) Test_Logout_ExpiresCookie() {
	res, err := hosting.ServeAuth(s.logoutRequest(s.host(true), http.MethodPost))

	s.NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusOK, res.Status)

	s.Require().Len(res.Cookies, 1)
	c := res.Cookies[0]
	s.Equal(buildconf.SessionCookieName, c.Name)
	s.Empty(c.Value)
	s.Equal(".example.com", c.Domain)
	s.Less(c.MaxAge, 0, "a deletion cookie must have MaxAge<0")
}

// Test_Logout_RejectsGet guards against a cross-site GET forcing a logout.
func (s *SkAuthCookieSuite) Test_Logout_RejectsGet() {
	res, err := hosting.ServeAuth(s.logoutRequest(s.host(true), http.MethodGet))

	s.NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusMethodNotAllowed, res.Status)
	s.Empty(res.Cookies)
}
