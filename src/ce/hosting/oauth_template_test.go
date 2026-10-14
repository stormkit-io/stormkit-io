package hosting_test

import (
	"errors"
	"html"
	"net/http"
	"net/url"
	"strings"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/mock"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	hosting "github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/mocks"
)

// oauthTemplate prints every field the OAuth page is rendered with, so a test
// can assert on the context rather than on markup.
const oauthTemplate = `{{if .error}}error={{.error}}{{else}}client={{.client_name}};` +
	`host={{.redirect_host}};signed_in={{.signed_in}};{{with .user}}email={{.email}};{{end}}` +
	`continue={{.continue_url}};cancel={{.cancel_url}};login={{.login_url}};` +
	`scopes={{range .scopes}}{{.name}}:{{.label}},{{end}}{{end}}`

// templateHost returns an OAuth-enabled host whose deployment ships
// /_stormkit/oauth.html with the given body.
func (s *OAuthSuite) templateHost(body string) *hosting.Host {
	client := &mocks.ClientInterface{}
	client.On("GetFile", integrations.GetFileArgs{
		Location:     "local:/deployments/1",
		DeploymentID: types.ID(1),
		FileName:     "/_stormkit/oauth.html",
	}).Return(&integrations.GetFileResult{Content: []byte(body)}, nil)

	integrations.SetDefaultClient(client)
	s.T().Cleanup(func() { integrations.SetDefaultClient(nil) })

	host := s.host(true)
	host.Config.DeploymentID = types.ID(1)
	host.Config.StorageLocation = "local:/deployments/1"
	host.Config.ReservedFiles = appconf.StaticFileConfig{
		"/_stormkit/oauth.html": {
			FileName: "/_stormkit/oauth.html",
			Headers:  map[string]string{"X-Custom": "yes", "Cache-Control": "public, max-age=3600"},
		},
	}

	return host
}

// page returns the rendered body with HTML escaping undone, so assertions can
// compare URLs as they were built.
func (s *OAuthSuite) page(res *shttp.Response) string {
	body, ok := res.Data.([]byte)
	s.Require().True(ok, "expected an HTML body")

	return html.UnescapeString(string(body))
}

// encryptedEmail encrypts email the way SkAuth stores it in the eml claim.
func encryptedEmail(email string) string {
	return utils.EncryptToString(email, []byte(oauthSecret))
}

// sessionWithEmail is session() with the given raw eml claim.
func (s *OAuthSuite) sessionWithEmail(uid, eml string) http.Header {
	tok, err := user.JWT(user.JWTParams{
		Purpose: user.PurposeSkAuthSession,
		Claims:  jwt.MapClaims{"uid": uid, "eml": eml},
		Secret:  oauthSecret,
	})

	s.Require().NoError(err)

	h := make(http.Header)
	h.Set("Cookie", buildconf.SessionCookieName+"="+tok)

	return h
}

func (s *OAuthSuite) authorizeWithTemplate(host *hosting.Host, query string, header http.Header) *shttp.Response {
	return hosting.HandleOAuthAuthorize(s.req(host, http.MethodGet, "/_stormkit/oauth/authorize", query, header, nil))
}

func (s *OAuthSuite) Test_Template_SignedIn() {
	query := s.authzQuery(pkce("verifier")) + "&scope=" + url.QueryEscape("openid offline_access")
	res := s.authorizeWithTemplate(s.templateHost(oauthTemplate), query, s.sessionWithEmail("user-1", encryptedEmail("jane@example.com")))
	page := s.page(res)

	s.Equal(http.StatusOK, res.Status)
	s.Contains(page, "client=chatgpt;")
	s.Contains(page, "host=chatgpt.com;")
	s.Contains(page, "signed_in=true;")
	s.Contains(page, "email=jane@example.com;")
	s.Contains(page, "continue=/_stormkit/oauth/authorize?"+query+";")
	s.Contains(page, "cancel="+oauthRedirect+"?error=access_denied&state=xyz;")
	s.Contains(page, "login=;")
	s.Contains(page, "scopes=openid:Verify your identity,offline_access:Maintain access when you are offline,")
}

// Test_Template_SignedIn_DecryptsEmail verifies that the page shows the email
// address, never the encrypted claim it is stored as.
func (s *OAuthSuite) Test_Template_SignedIn_DecryptsEmail() {
	eml := encryptedEmail("jane@example.com")
	res := s.authorizeWithTemplate(s.templateHost("{{.user.email}}"), s.authzQuery(pkce("verifier")), s.sessionWithEmail("user-1", eml))
	page := s.page(res)

	s.Equal(http.StatusOK, res.Status)
	s.Equal("jane@example.com", page)
	s.NotContains(page, eml)
}

// Test_Template_SignedIn_UndecryptableEmail verifies that a claim which cannot
// be decrypted renders an empty email while the visitor stays signed in.
func (s *OAuthSuite) Test_Template_SignedIn_UndecryptableEmail() {
	eml := utils.EncryptToString("jane@example.com", []byte("another-secret-padded-to-32-char"))
	res := s.authorizeWithTemplate(s.templateHost("{{.signed_in}};{{.user.email}};"), s.authzQuery(pkce("verifier")), s.sessionWithEmail("user-1", eml))
	page := s.page(res)

	s.Equal(http.StatusOK, res.Status)
	s.Equal("true;;", page)
}

func (s *OAuthSuite) Test_Template_SignedOut() {
	res := s.authorizeWithTemplate(s.templateHost(oauthTemplate), s.authzQuery(pkce("verifier")), nil)
	page := s.page(res)

	s.Equal(http.StatusOK, res.Status)
	s.Nil(res.Redirect)
	s.Contains(page, "signed_in=false;")
	s.NotContains(page, "email=")
	s.Contains(page, "continue=;")
	s.Contains(page, "login=;")
}

// Test_Template_SignedOut_LoginURL verifies that a configured login URL is
// handed to the page instead of being redirected to.
func (s *OAuthSuite) Test_Template_SignedOut_LoginURL() {
	host := s.templateHost(oauthTemplate)
	host.Config.SKAuth.LoginURL = "/login"

	res := s.authorizeWithTemplate(host, s.authzQuery(pkce("verifier")), nil)
	page := s.page(res)

	s.Equal(http.StatusOK, res.Status)
	s.Nil(res.Redirect)
	s.Contains(page, "login=https://app.example.com/login?return_to=")
}

func (s *OAuthSuite) Test_Template_Headers() {
	res := s.authorizeWithTemplate(s.templateHost(oauthTemplate), s.authzQuery(pkce("verifier")), nil)

	s.Equal("text/html; charset=utf-8", res.Headers.Get("Content-Type"))
	s.Equal("same-origin", res.Headers.Get("Referrer-Policy"))
	s.Equal("no-store", res.Headers.Get("Cache-Control"))
	s.Equal("DENY", res.Headers.Get("X-Frame-Options"))
	s.Equal("frame-ancestors 'none'", res.Headers.Get("Content-Security-Policy"))
	s.Equal("yes", res.Headers.Get("X-Custom"))
}

// Test_Template_Error verifies that a request that cannot go on is explained on
// the app's page, without a cancel link to an unvalidated redirect_uri.
func (s *OAuthSuite) Test_Template_Error() {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", "evil")
	q.Set("redirect_uri", "https://evil.example.com/cb")

	res := s.authorizeWithTemplate(s.templateHost(oauthTemplate), q.Encode(), nil)
	page := s.page(res)

	s.Equal(http.StatusBadRequest, res.Status)
	s.Equal("error=The redirect_uri is not allowed for this application.", page)
	s.Equal("DENY", res.Headers.Get("X-Frame-Options"))
}

func (s *OAuthSuite) Test_Template_EscapesClientName() {
	id := s.registerClient("<script>alert(1)</script>", oauthRedirect)
	host := s.templateHost(`<p>{{.client_name}}</p>`)

	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", id)
	q.Set("redirect_uri", oauthRedirect)
	q.Set("code_challenge", pkce("verifier"))
	q.Set("code_challenge_method", "S256")

	res := s.authorizeWithTemplate(host, q.Encode(), nil)

	s.Equal(http.StatusOK, res.Status)
	s.Equal("<p>&lt;script&gt;alert(1)&lt;/script&gt;</p>", string(res.Data.([]byte)))
}

// Test_Template_ParseError_FallsBack shows the template error on the built-in
// page rather than a half-rendered one.
func (s *OAuthSuite) Test_Template_ParseError_FallsBack() {
	res := s.authorizeWithTemplate(s.templateHost(`{{if .signed_in}}`), s.authzQuery(pkce("verifier")), nil)

	s.Equal(http.StatusBadRequest, res.Status)
	s.Contains(s.page(res), "could not be rendered")
}

func (s *OAuthSuite) Test_Template_LoadError_FallsBack() {
	host := s.templateHost(oauthTemplate)

	client := &mocks.ClientInterface{}
	client.On("GetFile", mock.Anything).Return(nil, errors.New("storage down"))
	integrations.SetDefaultClient(client)

	res := s.authorizeWithTemplate(host, s.authzQuery(pkce("verifier")), nil)

	s.Equal(http.StatusBadRequest, res.Status)
	s.Contains(s.page(res), "could not be loaded")
}

func (s *OAuthSuite) formGrant(host *hosting.Host, header http.Header) *shttp.Response {
	header.Set("Content-Type", "application/x-www-form-urlencoded")

	return hosting.HandleOAuthGrant(s.req(host, http.MethodPost, "/_stormkit/oauth/authorize", s.authzQuery(pkce("verifier")), header, strings.NewReader("")))
}

func (s *OAuthSuite) Test_Grant_Form_RedirectsWithCode() {
	h := s.session("user-1")
	h.Set("Origin", "https://app.example.com")

	res := s.formGrant(s.templateHost(oauthTemplate), h)

	s.Require().Equal(http.StatusFound, res.Status)
	s.Require().NotNil(res.Redirect)

	u, err := url.Parse(*res.Redirect)
	s.Require().NoError(err)
	s.Equal(oauthRedirect, u.Scheme+"://"+u.Host+u.Path)
	s.NotEmpty(u.Query().Get("code"))
	s.Equal("xyz", u.Query().Get("state"))
}

// Test_Grant_Form_KeepsEmailEncrypted verifies that the access token carries the
// session's eml claim still encrypted, as the edge middleware expects it.
func (s *OAuthSuite) Test_Grant_Form_KeepsEmailEncrypted() {
	eml := encryptedEmail("jane@example.com")
	h := s.sessionWithEmail("user-1", eml)
	h.Set("Origin", "https://app.example.com")

	res := s.formGrant(s.templateHost(oauthTemplate), h)
	s.Require().Equal(http.StatusFound, res.Status)

	u, err := url.Parse(*res.Redirect)
	s.Require().NoError(err)

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", u.Query().Get("code"))
	form.Set("client_id", "chatgpt")
	form.Set("redirect_uri", oauthRedirect)
	form.Set("code_verifier", "verifier")

	token := hosting.HandleOAuthToken(s.tokenReq(form))
	s.Require().Equal(http.StatusOK, token.Status)

	accessToken, _ := s.json(token)["access_token"].(string)
	claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: accessToken, Secret: oauthSecret, MaxMins: 10, Purposes: []user.Purpose{user.PurposeSkAuthAccessToken}})
	s.Require().NotNil(claims)
	s.Equal(eml, claims["eml"])
}

// Test_Grant_Form_RejectsNullOrigin covers a form posted under a no-referrer
// policy, which browsers send with "Origin: null".
func (s *OAuthSuite) Test_Grant_Form_RejectsNullOrigin() {
	h := s.session("user-1")
	h.Set("Origin", "null")

	res := s.formGrant(s.templateHost(oauthTemplate), h)

	s.Equal(http.StatusForbidden, res.Status)
	s.Nil(res.Redirect)
	s.Contains(s.page(res), "error=cross-origin request rejected")
}

func (s *OAuthSuite) Test_Grant_Form_NoSession() {
	res := s.formGrant(s.templateHost(oauthTemplate), make(http.Header))

	s.Equal(http.StatusUnauthorized, res.Status)
	s.Contains(s.page(res), "error=authentication required")
}

// Test_Grant_Form_BuiltInPage covers a form post to an app without its own page.
func (s *OAuthSuite) Test_Grant_Form_BuiltInPage() {
	res := s.formGrant(s.host(true), make(http.Header))

	s.Equal(http.StatusUnauthorized, res.Status)
	s.Contains(s.page(res), "authentication required")
}
