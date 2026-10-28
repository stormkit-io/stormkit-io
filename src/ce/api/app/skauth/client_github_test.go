package skauth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stretchr/testify/suite"
	"golang.org/x/oauth2"
)

type ClientGitHubSuite struct {
	suite.Suite

	server       *httptest.Server
	originalBase string
	emails       string
}

func (s *ClientGitHubSuite) SetupTest() {
	s.originalBase = skauth.GitHubAPIBase
	s.emails = `[
		{"email": "octocat@users.noreply.github.com", "primary": false, "verified": true},
		{"email": "octo@work.test", "primary": false, "verified": true},
		{"email": "octocat@home.test", "primary": true, "verified": true}
	]`

	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("Bearer test-access-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/user":
			w.Write([]byte(`{"id": 583231, "login": "octocat", "name": "The Octocat", "avatar_url": "https://avatars.test/octocat", "html_url": "https://github.com/octocat"}`))
		case "/user/emails":
			w.Write([]byte(s.emails))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	skauth.GitHubAPIBase = s.server.URL
}

func (s *ClientGitHubSuite) TearDownTest() {
	skauth.GitHubAPIBase = s.originalBase
	s.server.Close()
}

func (s *ClientGitHubSuite) userInfo() (*skauth.UserInfo, error) {
	client := skauth.NewGitHubClient("client-id", "client-secret", "https://app.example.com/_stormkit/auth/callback")
	return client.UserInfo(context.Background(), &oauth2.Token{AccessToken: "test-access-token", TokenType: "Bearer"})
}

func (s *ClientGitHubSuite) Test_UserInfo_PrimaryVerifiedEmail() {
	info, err := s.userInfo()

	s.Require().NoError(err)
	s.Equal("583231", info.AccountID)
	s.Equal("octocat@home.test", info.Email)
	s.True(info.EmailVerified)
	s.Equal("The Octocat", info.FirstName)
	s.Equal("https://avatars.test/octocat", info.Avatar)
	s.Equal("octocat", info.Username)
	s.Equal("https://github.com/octocat", info.ProfileURL)
}

// Test_UserInfo_UnverifiedPrimary verifies that an unverified primary email
// is never used, and a verified secondary one is picked instead.
func (s *ClientGitHubSuite) Test_UserInfo_UnverifiedPrimary() {
	s.emails = `[
		{"email": "octocat@home.test", "primary": true, "verified": false},
		{"email": "octo@work.test", "primary": false, "verified": true}
	]`

	info, err := s.userInfo()

	s.Require().NoError(err)
	s.Equal("octo@work.test", info.Email)
	s.True(info.EmailVerified)
}

// Test_UserInfo_OnlyRelayAddress verifies that GitHub's relay address does
// not count as a verified email, so the callback refuses the sign-in.
func (s *ClientGitHubSuite) Test_UserInfo_OnlyRelayAddress() {
	s.emails = `[{"email": "583231+octocat@users.noreply.github.com", "primary": true, "verified": true}]`

	info, err := s.userInfo()

	s.Require().NoError(err)
	s.Empty(info.Email)
	s.False(info.EmailVerified)
}

func (s *ClientGitHubSuite) Test_UserInfo_ApiError() {
	s.emails = `not json`

	info, err := s.userInfo()

	s.Error(err)
	s.Nil(info)
}

func (s *ClientGitHubSuite) Test_AuthCodeURL() {
	client := skauth.NewGitHubClient("client-id", "client-secret", "https://app.example.com/_stormkit/auth/callback")

	authURL, err := client.AuthCodeURL(skauth.AuthCodeURLParams{EnvID: 1, ProviderName: skauth.ProviderGitHub, Secret: stateSecret})
	s.Require().NoError(err)

	parsed, err := url.Parse(authURL)
	s.Require().NoError(err)
	s.Equal("github.com", parsed.Host)
	s.Equal("client-id", parsed.Query().Get("client_id"))
	s.Equal("https://app.example.com/_stormkit/auth/callback", parsed.Query().Get("redirect_uri"))
	s.Equal("read:user user:email", parsed.Query().Get("scope"))

	claims := user.ParseJWT(&user.ParseJWTArgs{
		Bearer:   parsed.Query().Get("state"),
		Secret:   stateSecret,
		Purposes: []user.Purpose{user.PurposeSkAuthOAuthState},
	})

	s.Equal(skauth.ProviderGitHub, claims["prv"])
}

func (s *ClientGitHubSuite) Test_ProviderClient() {
	p := &skauth.Provider{Name: skauth.ProviderGitHub}

	s.True(p.Supported())
	s.IsType(&skauth.GitHubClient{}, p.Client(skauth.ClientParams{RedirectURL: "https://app.example.com/_stormkit/auth/callback"}))
}

func TestClientGitHubSuite(t *testing.T) {
	suite.Run(t, new(ClientGitHubSuite))
}
