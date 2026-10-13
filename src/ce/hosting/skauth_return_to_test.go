package hosting_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/mock"
	"golang.org/x/oauth2"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/mocks"
)

// authorizeReturnTo is the OAuth page a sign-in is started from in these tests.
const authorizeReturnTo = "/_stormkit/oauth/authorize?client_id=chatgpt&state=xyz"

// withOAuthServer enables the OAuth server on host, which return_to requires.
func withOAuthServer(host *hosting.Host) *hosting.Host {
	host.Config.SKAuth.Status = true
	host.Config.SKAuth.OAuthServer = &buildconf.OAuthServerConf{Enabled: true}

	return host
}

// formPost builds an HTML form submission to target.
func formPost(host *hosting.Host, target string, form url.Values) *hosting.RequestContext {
	u, err := url.Parse(target)

	if err != nil {
		panic(err)
	}

	u.Scheme = "https"
	u.Host = host.Name

	rq := &hosting.RequestContext{
		Host: host,
		RequestContext: shttp.NewRequestContext(&http.Request{
			Method: http.MethodPost,
			Header: http.Header{"Content-Type": {"application/x-www-form-urlencoded"}},
			Body:   io.NopCloser(strings.NewReader(form.Encode())),
			URL:    u,
		}),
	}

	rq.OriginalPath = u.Path

	return rq
}

func (s *WithSKAuthMagicLinkSuite) requestFromOAuthPage(host *hosting.Host, email, returnTo string) *shttp.Response {
	target := "/_stormkit/auth/magic?return_to=" + url.QueryEscape(returnTo)
	res, err := hosting.ServeAuth(formPost(host, target, url.Values{"email": {email}}))
	s.Require().NoError(err)

	return res
}

func (s *WithSKAuthMagicLinkSuite) Test_ReturnTo_LinkSent_ThenLandsOnOAuthPage() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	host := withOAuthServer(s.hostFor(env.ID))
	res := s.requestFromOAuthPage(host, "user@example.com", authorizeReturnTo)

	s.Equal(http.StatusSeeOther, res.Status)
	s.Require().NotNil(res.Redirect)
	s.Equal("/_stormkit/oauth/authorize?client_id=chatgpt&sk_sign_in=link_sent&state=xyz", *res.Redirect)

	verify, err := hosting.ServeAuth(s.magicRequest(host, "/_stormkit/auth/magic?token="+s.captureToken(env.ID)))
	s.Require().NoError(err)

	s.Equal(http.StatusFound, verify.Status)
	s.Require().NotNil(verify.Redirect)
	s.Equal(authorizeReturnTo, *verify.Redirect)
	s.Require().Len(verify.Cookies, 1)
	s.Equal(buildconf.SessionCookieName, verify.Cookies[0].Name)
}

// Test_ReturnTo_AbsoluteSameHost accepts the full authorize URL a page may
// build from window.location, and strips a stale outcome from it.
func (s *WithSKAuthMagicLinkSuite) Test_ReturnTo_AbsoluteSameHost() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	host := withOAuthServer(s.hostFor(env.ID))
	res := s.requestFromOAuthPage(host, "user@example.com", "https://my.example.com"+authorizeReturnTo+"&sk_sign_in=expired")

	s.Require().NotNil(res.Redirect)
	s.Equal("/_stormkit/oauth/authorize?client_id=chatgpt&sk_sign_in=link_sent&state=xyz", *res.Redirect)
}

func (s *WithSKAuthMagicLinkSuite) Test_ReturnTo_InvalidEmail() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	res := s.requestFromOAuthPage(withOAuthServer(s.hostFor(env.ID)), "not-an-email", authorizeReturnTo)

	s.Equal(http.StatusSeeOther, res.Status)
	s.Require().NotNil(res.Redirect)
	s.Contains(*res.Redirect, "sk_sign_in=invalid_email")
}

// Test_ReturnTo_Rejected verifies that only this host's authorize endpoint is
// accepted: anything else is ignored and the request behaves as before.
func (s *WithSKAuthMagicLinkSuite) Test_ReturnTo_Rejected() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	host := withOAuthServer(s.hostFor(env.ID))

	for _, returnTo := range []string{
		"https://evil.example.com" + authorizeReturnTo,
		"//evil.example.com" + authorizeReturnTo,
		"http://my.example.com" + authorizeReturnTo,
		"/dashboard",
		"/_stormkit/oauth/authorize/../../dashboard",
		"javascript:alert(1)",
	} {
		res := s.requestFromOAuthPage(host, "user@example.com", returnTo)

		s.Equal(http.StatusCreated, res.Status, returnTo)
		s.Nil(res.Redirect, returnTo)
	}

	emails, err := buildconf.MailerStore().Emails(context.Background(), env.ID)
	s.Require().NoError(err)
	s.Require().Len(emails, 6)

	for _, email := range emails {
		start := strings.Index(email.Body, "token=") + len("token=")
		token := email.Body[start : start+strings.IndexAny(email.Body[start:], `"<> `)]
		claims := jwt.MapClaims{}

		_, _, err := jwt.NewParser().ParseUnverified(token, claims)
		s.Require().NoError(err)
		s.NotContains(claims, "rto")
	}
}

func (s *WithSKAuthMagicLinkSuite) Test_ReturnTo_IgnoredWhenOAuthServerDisabled() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	res := s.requestFromOAuthPage(s.hostFor(env.ID), "user@example.com", authorizeReturnTo)

	s.Equal(http.StatusCreated, res.Status)
	s.Nil(res.Redirect)
}

// Test_ReturnTo_ExpiredLink sends an expired link back to the OAuth page it was
// requested from, so the visitor can ask for a new one there.
func (s *WithSKAuthMagicLinkSuite) Test_ReturnTo_ExpiredLink() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	expired, err := user.JWT(user.JWTParams{Purpose: user.PurposeSkAuthMagicLink, Claims: jwt.MapClaims{
		"exp": time.Now().Add(-time.Hour).Unix(),
		"prv": skauth.ProviderMagicLink,
		"rto": authorizeReturnTo,
	}, Secret: env.AuthConf.Secret})
	s.Require().NoError(err)

	res, err := hosting.ServeAuth(s.magicRequest(withOAuthServer(s.hostFor(env.ID)), "/_stormkit/auth/magic?token="+expired))
	s.Require().NoError(err)

	s.Equal(http.StatusSeeOther, res.Status)
	s.Require().NotNil(res.Redirect)
	s.Equal("/_stormkit/oauth/authorize?client_id=chatgpt&sk_sign_in=expired&state=xyz", *res.Redirect)
	s.Empty(res.Cookies)
}

func (s *WithSKAuthOAuthSuite) Test_Initiate_ReturnTo_RidesState() {
	env := s.setupEnv([]string{"https://app.example.com"}, true)

	s.mockClient.On("AuthCodeURL", skauth.AuthCodeURLParams{
		EnvID:        env.ID,
		ProviderName: skauth.ProviderGoogle,
		Referrer:     "https://api.example.com",
		Secret:       "test-secret-padded-to-32-chars!!",
		ReturnTo:     authorizeReturnTo,
	}).Return("https://accounts.google.com/o/oauth2/auth", nil)

	res, err := hosting.ServeAuth(s.oauthRequest(
		withOAuthServer(s.hostFor(env.ID)),
		"/_stormkit/auth/google?redirect=https://app.example.com&return_to="+url.QueryEscape(authorizeReturnTo),
		nil,
	))

	s.Require().NoError(err)
	s.Equal(http.StatusFound, res.Status)
	s.mockClient.AssertExpectations(s.T())
}

func (s *WithSKAuthOAuthSuite) stateWithReturnTo(returnTo string) string {
	token, err := user.JWT(user.JWTParams{Purpose: user.PurposeSkAuthOAuthState, Claims: jwt.MapClaims{
		"prv": skauth.ProviderGoogle,
		"ref": "https://api.example.com",
		"rto": returnTo,
	}, Secret: "test-secret-padded-to-32-chars!!"})

	s.Require().NoError(err)

	return token
}

func (s *WithSKAuthOAuthSuite) Test_Callback_ReturnTo_LandsOnOAuthPage() {
	host := withOAuthServer(s.setupCallbackEnv(true))
	token := &oauth2.Token{}

	s.mockClient.On("Exchange", mock.Anything, mock.Anything).Return(token, nil).Once()
	s.mockClient.On("UserInfo", mock.Anything, token).Return(&skauth.UserInfo{
		AccountID:     "test-account-id",
		Email:         "jane@stormkit.io",
		EmailVerified: true,
	}, nil).Once()

	res, err := hosting.ServeAuth(s.oauthRequest(
		host,
		fmt.Sprintf("/_stormkit/auth/callback?state=%s&code=test-code", s.stateWithReturnTo(authorizeReturnTo)),
		nil,
	))

	s.Require().NoError(err)
	s.Equal(http.StatusFound, res.Status)
	s.Require().NotNil(res.Redirect)
	s.Equal(authorizeReturnTo, *res.Redirect)
	s.Require().Len(res.Cookies, 1)
}

func (s *WithSKAuthOAuthSuite) Test_Callback_ReturnTo_Cancelled() {
	host := withOAuthServer(s.setupCallbackEnv(true))

	res, err := hosting.ServeAuth(s.oauthRequest(
		host,
		fmt.Sprintf("/_stormkit/auth/callback?state=%s&error=access_denied", s.stateWithReturnTo(authorizeReturnTo)),
		nil,
	))

	s.Require().NoError(err)
	s.Equal(http.StatusSeeOther, res.Status)
	s.Require().NotNil(res.Redirect)
	s.Equal("/_stormkit/oauth/authorize?client_id=chatgpt&sk_sign_in=failed&state=xyz", *res.Redirect)
}

// Test_Callback_ReturnTo_Forged ignores a return_to that is not this host's
// authorize endpoint, even inside a validly signed state.
func (s *WithSKAuthOAuthSuite) Test_Callback_ReturnTo_Forged() {
	host := withOAuthServer(s.setupCallbackEnv(true))

	res, err := hosting.ServeAuth(s.oauthRequest(
		host,
		fmt.Sprintf("/_stormkit/auth/callback?state=%s&error=access_denied", s.stateWithReturnTo("https://evil.example.com/_stormkit/oauth/authorize")),
		nil,
	))

	s.Require().NoError(err)
	s.Require().NotNil(res.Redirect)
	s.Contains(*res.Redirect, "https://api.example.com/dashboard?login_error=")
}

// Test_Template_SignInLinks verifies the OAuth page is given a link per enabled
// provider, each returning to the authorize request it was rendered for.
func (s *WithSKAuthOAuthSuite) Test_Template_SignInLinks() {
	host := withOAuthServer(s.setupCallbackEnv(true))
	host.Config.DeploymentID = types.ID(1)
	host.Config.StorageLocation = "local:/deployments/1"
	host.Config.ReservedFiles = appconf.StaticFileConfig{
		"/_stormkit/oauth.html": {FileName: "/_stormkit/oauth.html"},
	}

	client := &mocks.ClientInterface{}
	client.On("GetFile", mock.Anything).Return(&integrations.GetFileResult{
		Content: []byte(`google={{.sign_in.google}};x={{.sign_in.x}};magic={{.sign_in.magic_link}};error={{.sign_in.error}}`),
	}, nil)

	integrations.SetDefaultClient(client)
	defer integrations.SetDefaultClient(nil)

	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", "chatgpt")
	q.Set("redirect_uri", "https://chatgpt.com/connector/oauth/cb")
	q.Set("code_challenge", "challenge")
	q.Set("code_challenge_method", "S256")
	q.Set("sk_sign_in", "failed")
	q.Set("sk_auth_retried", "1")

	res := hosting.HandleOAuthAuthorize(s.oauthRequest(host, "/_stormkit/oauth/authorize?"+q.Encode(), nil))
	s.Require().Equal(http.StatusOK, res.Status)

	q.Del("sk_sign_in")
	q.Del("sk_auth_retried")

	returnTo := url.QueryEscape("/_stormkit/oauth/authorize?" + q.Encode())
	body := strings.ReplaceAll(string(res.Data.([]byte)), "&amp;", "&")

	s.Contains(body, "google=/_stormkit/auth/google?return_to="+returnTo+";")
	s.Contains(body, "x=;")
	s.Contains(body, "magic=;")
	s.Contains(body, "error=Sign-in did not complete. Please try again.")
}
