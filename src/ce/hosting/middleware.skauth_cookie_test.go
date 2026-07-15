package hosting_test

import (
	"net/http"
	"net/url"
	"testing"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
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

func (s *SkAuthCookieSuite) host() *hosting.Host {
	return &hosting.Host{
		Name: "www.example.com",
		Config: &appconf.Config{SKAuth: &buildconf.SKAuthConf{
			Secret:       cookieSecret,
			SuccessURL:   "/",
			Status:       true,
			TTL:          10,
			CookieDomain: ".example.com",
		}},
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

	req := s.request(s.host(), h)
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

	req := s.request(s.host(), h)
	_, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.Equal("user-7", req.Header.Get("X-User-Id"))
}

// Test_Edge_NoCredential_NoIdentity verifies no session yields no identity
// header (and any client-supplied one is stripped).
func (s *SkAuthCookieSuite) Test_Edge_NoCredential_NoIdentity() {
	h := make(http.Header)
	h.Set("X-User-Id", "spoofed")

	req := s.request(s.host(), h)
	_, err := hosting.WithSKAuth(req)

	s.NoError(err)
	s.Empty(req.Header.Get("X-User-Id"))
}

// refreshRequest builds a POST /_stormkit/auth/refresh with the given headers.
func (s *SkAuthCookieSuite) refreshRequest(host *hosting.Host, header http.Header) *hosting.RequestContext {
	req := s.request(host, header)
	req.RequestContext.Method = http.MethodPost
	req.RequestContext.Request.URL = &url.URL{Host: host.Name, Path: "/_stormkit/auth/refresh", RawPath: "/_stormkit/auth/refresh"}
	req.OriginalPath = "/_stormkit/auth/refresh"

	return req
}

// Test_Refresh_CookieMode_BearerDelivery verifies a native/mobile client can
// rotate its session on a cookie-mode env: authenticating with an Authorization
// bearer and opting into bearer delivery yields a new token in the body and no
// cookie.
func (s *SkAuthCookieSuite) Test_Refresh_CookieMode_BearerDelivery() {
	h := make(http.Header)
	h.Set("Authorization", "Bearer "+s.token("user-mobile"))
	h.Set("X-Session-Delivery", "bearer")

	res, err := hosting.ServeAuth(s.refreshRequest(s.host(), h))

	s.NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusOK, res.Status)

	data, ok := res.Data.(map[string]any)
	s.Require().True(ok)
	s.NotEmpty(data["token"], "bearer delivery must return a rotated token")
	s.Empty(res.Cookies, "bearer delivery must not set a cookie")
}

// Test_Refresh_CookieMode_CookieDelivery verifies the browser path: a
// cookie-authenticated refresh rotates the cookie and returns no token.
func (s *SkAuthCookieSuite) Test_Refresh_CookieMode_CookieDelivery() {
	h := make(http.Header)
	h.Set("Cookie", buildconf.SessionCookieName+"="+s.token("user-browser"))
	h.Set("Origin", "https://www.example.com")

	res, err := hosting.ServeAuth(s.refreshRequest(s.host(), h))

	s.NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusOK, res.Status)

	data, ok := res.Data.(map[string]any)
	s.Require().True(ok)
	_, hasToken := data["token"]
	s.False(hasToken, "cookie delivery must not return the token in the body")

	s.Require().Len(res.Cookies, 1)
	s.Equal(buildconf.SessionCookieName, res.Cookies[0].Name)
}

// Test_Logout_ExpiresCookie verifies POST /_stormkit/auth/logout clears the
// session cookie — the only way to end a session, since the HttpOnly cookie is
// invisible to the app's JS.
func (s *SkAuthCookieSuite) Test_Logout_ExpiresCookie() {
	req := s.request(s.host(), nil)
	req.RequestContext.Method = http.MethodPost
	req.RequestContext.Request.URL = &url.URL{Host: "www.example.com", Path: "/_stormkit/auth/logout", RawPath: "/_stormkit/auth/logout"}
	req.OriginalPath = "/_stormkit/auth/logout"

	res, err := hosting.ServeAuth(req)

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
