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

// triggerHeader returns request headers carrying a trigger token of envID.
func (s *AuthWallMiddlewareSuite) triggerHeader(envID types.ID) http.Header {
	token, err := authwall.Token{EnvID: envID}.Trigger(types.ID(42))
	s.Require().NoError(err)

	h := make(http.Header)
	h.Set(authwall.HeaderTriggerToken, token)
	h.Set(hosting.HeaderAuthWallTrigger, "forged")

	return h
}

// Test_TriggerToken_OpensWall verifies that a periodic trigger's request gets
// through the wall, and that the app learns which trigger sent it but never
// sees the token.
func (s *AuthWallMiddlewareSuite) Test_TriggerToken_OpensWall() {
	req := s.request("all", s.triggerHeader(types.ID(5)))
	res, err := hosting.WithAuthWall(req)

	s.NoError(err)
	s.Nil(res)
	s.Equal("42", req.Header.Get(hosting.HeaderAuthWallTrigger))
	s.Empty(req.Header.Get(authwall.HeaderTriggerToken))
	s.Empty(req.Header.Get(hosting.HeaderAuthWallEmail))
}

// Test_TriggerToken_OtherEnv verifies that a trigger of one environment cannot
// open the wall of another.
func (s *AuthWallMiddlewareSuite) Test_TriggerToken_OtherEnv() {
	req := s.request("all", s.triggerHeader(types.ID(6)))
	res, err := hosting.WithAuthWall(req)

	s.NoError(err)
	s.Require().NotNil(res)
	s.Contains(string(res.Data.([]byte)), "Password protected deployment")
	s.Empty(req.Header.Get(hosting.HeaderAuthWallTrigger))
}

// Test_TriggerToken_SessionTokenRejected verifies that a visitor's session
// cannot be passed off as a trigger token.
func (s *AuthWallMiddlewareSuite) Test_TriggerToken_SessionTokenRejected() {
	session, err := authwall.Token{EnvID: types.ID(5)}.Session(authwall.Session{})
	s.Require().NoError(err)

	h := make(http.Header)
	h.Set(authwall.HeaderTriggerToken, session)

	req := s.request("all", h)
	res, err := hosting.WithAuthWall(req)

	s.NoError(err)
	s.NotNil(res)
	s.Empty(req.Header.Get(hosting.HeaderAuthWallTrigger))
}

// Test_TriggerToken_StrippedWithoutWall verifies that the app never receives
// the token or a forged trigger header, whether or not the wall is on.
func (s *AuthWallMiddlewareSuite) Test_TriggerToken_StrippedWithoutWall() {
	for _, mode := range []string{"", "dev"} {
		req := s.request(mode, s.triggerHeader(types.ID(5)))
		_, err := hosting.WithAuthWall(req)

		s.NoError(err)
		s.Empty(req.Header.Get(authwall.HeaderTriggerToken), mode)
		s.Empty(req.Header.Get(hosting.HeaderAuthWallTrigger), mode)
	}
}
