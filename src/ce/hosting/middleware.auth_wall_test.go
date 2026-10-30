package hosting_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type AuthWallMiddlewareSuite struct {
	suite.Suite
}

func TestAuthWallMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(AuthWallMiddlewareSuite))
}

// request builds a request to a host whose Auth Wall is in the given mode.
func (s *AuthWallMiddlewareSuite) request(authWall string, header http.Header) *hosting.RequestContext {
	host := &hosting.Host{
		Name:   "www.example.com",
		Config: &appconf.Config{EnvID: types.ID(5), AuthWall: authWall},
	}

	return &hosting.RequestContext{
		Host: host,
		RequestContext: shttp.NewRequestContext(&http.Request{
			Method: http.MethodGet,
			Header: header,
			URL:    &url.URL{Host: host.Name, Path: "/"},
		}),
	}
}

// forgedHeader returns request headers carrying a forged login.
func (s *AuthWallMiddlewareSuite) forgedHeader() http.Header {
	h := make(http.Header)
	h.Set(hosting.HeaderAuthWallEmail, "forged@example.org")
	h.Set(hosting.HeaderAuthWallMetadata, "admin")

	return h
}

// Test_PassesLoginToApp verifies that a logged-in visitor's email and metadata
// reach the app, replacing any the visitor sent.
func (s *AuthWallMiddlewareSuite) Test_PassesLoginToApp() {
	token, err := authwall.Token{EnvID: types.ID(5)}.Session(authwall.Session{
		Email:    "client@example.org",
		Metadata: `{"tenant":"acme"}`,
	})
	s.Require().NoError(err)

	h := s.forgedHeader()
	h.Set("Cookie", hosting.SESSION_COOKIE_NAME+"="+token)

	req := s.request("all", h)
	res, err := hosting.WithAuthWall(req)

	s.NoError(err)
	s.Nil(res)
	s.Equal("client@example.org", req.Header.Get(hosting.HeaderAuthWallEmail))
	s.Equal(`{"tenant":"acme"}`, req.Header.Get(hosting.HeaderAuthWallMetadata))
}

// Test_StripsForgedHeaders verifies that the app never receives login headers
// the visitor sent, whether or not the wall is on.
func (s *AuthWallMiddlewareSuite) Test_StripsForgedHeaders() {
	for _, mode := range []string{"", "dev", "all"} {
		req := s.request(mode, s.forgedHeader())
		_, err := hosting.WithAuthWall(req)

		s.NoError(err)
		s.Empty(req.Header.Get(hosting.HeaderAuthWallEmail), mode)
		s.Empty(req.Header.Get(hosting.HeaderAuthWallMetadata), mode)
	}
}

// Test_SessionWithoutMetadata verifies that a session issued before logins had
// metadata still opens the wall, without login headers.
func (s *AuthWallMiddlewareSuite) Test_SessionWithoutMetadata() {
	token, err := authwall.Token{EnvID: types.ID(5)}.Session(authwall.Session{})
	s.Require().NoError(err)

	h := make(http.Header)
	h.Set("Cookie", hosting.SESSION_COOKIE_NAME+"="+token)

	req := s.request("all", h)
	res, err := hosting.WithAuthWall(req)

	s.NoError(err)
	s.Nil(res)
	s.Empty(req.Header.Get(hosting.HeaderAuthWallEmail))
}

// publicRequest builds a request to path on a protected host whose wall
// leaves /api/webhooks/* public.
func (s *AuthWallMiddlewareSuite) publicRequest(path string, header http.Header) *hosting.RequestContext {
	req := s.request("all", header)
	req.Host.Config.AuthWallPublicPaths = []string{"/api/webhooks/*"}
	req.Request.URL.Path = path

	return req
}

// Test_PublicPath_SkipsWall verifies that public paths are served without a
// login, and that the app still cannot be handed a forged login on them.
func (s *AuthWallMiddlewareSuite) Test_PublicPath_SkipsWall() {
	req := s.publicRequest("/api/webhooks/stripe", s.forgedHeader())
	res, err := hosting.WithAuthWall(req)

	s.NoError(err)
	s.Nil(res)
	s.Empty(req.Header.Get(hosting.HeaderAuthWallEmail))
}

// Test_PublicPath_KeepsLogin verifies that a logged-in visitor's login still
// reaches the app on a public path.
func (s *AuthWallMiddlewareSuite) Test_PublicPath_KeepsLogin() {
	token, err := authwall.Token{EnvID: types.ID(5)}.Session(authwall.Session{Email: "client@example.org"})
	s.Require().NoError(err)

	h := make(http.Header)
	h.Set("Cookie", hosting.SESSION_COOKIE_NAME+"="+token)

	req := s.publicRequest("/api/webhooks/stripe", h)
	res, err := hosting.WithAuthWall(req)

	s.NoError(err)
	s.Nil(res)
	s.Equal("client@example.org", req.Header.Get(hosting.HeaderAuthWallEmail))
}

// Test_PublicPath_OtherPathsProtected verifies that paths outside the public
// ones, including ones that climb out of them, still require a login.
func (s *AuthWallMiddlewareSuite) Test_PublicPath_OtherPathsProtected() {
	for _, path := range []string{"/", "/api/webhooks", "/api/webhooks/../admin"} {
		res, err := hosting.WithAuthWall(s.publicRequest(path, make(http.Header)))

		s.NoError(err)
		s.NotNil(res, path)
	}
}
