package hosting_test

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	hosting "github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

const oauthSecret = "test-secret-padded-to-32-chars!!"

// oauthRedirect is a curated connector preset origin (ChatGPT's dynamic callback
// path), which the OAuth server trusts automatically once enabled. Redirect
// validation is preset-based, not AllowedOrigins-based, so tests exercise the
// real connector path.
const oauthRedirect = "https://chatgpt.com/connector/oauth/cb"

type OAuthSuite struct {
	suite.Suite
}

func TestOAuthSuite(t *testing.T) {
	suite.Run(t, new(OAuthSuite))
}

// SetupTest clears the shared host's registration rate-limit counter so a
// registration test never trips the limit because of a prior test or re-run
// against the same Redis.
func (s *OAuthSuite) SetupTest() {
	hosting.ResetOAuthRateLimit("app.example.com")
}

func (s *OAuthSuite) host(enabled bool) *hosting.Host {
	conf := &buildconf.SKAuthConf{
		Secret:         oauthSecret,
		Status:         true,
		TTL:            10,
		AllowedOrigins: []string{"https://client.example.com"},
		OAuthServer:    &buildconf.OAuthServerConf{Enabled: enabled},
	}

	return &hosting.Host{
		Name:   "app.example.com",
		Config: &appconf.Config{SKAuth: conf},
	}
}

func (s *OAuthSuite) req(host *hosting.Host, method, path, query string, header http.Header, body io.Reader) *hosting.RequestContext {
	if header == nil {
		header = make(http.Header)
	}

	var rc io.ReadCloser

	if body != nil {
		rc = io.NopCloser(body)
	}

	rq := &hosting.RequestContext{
		Host: host,
		RequestContext: shttp.NewRequestContext(&http.Request{
			Method: method,
			Header: header,
			URL:    &url.URL{Host: host.Name, Path: path, RawPath: path, RawQuery: query},
			Body:   rc,
		}),
	}

	rq.OriginalPath = path

	return rq
}

// bearer signs a SkAuth session token the way the login handlers do.
func (s *OAuthSuite) bearer(uid string) string {
	tok, err := user.JWT(jwt.MapClaims{"uid": uid}, oauthSecret)
	s.Require().NoError(err)

	return "Bearer " + tok
}

// session returns request headers carrying a valid SkAuth session cookie, as a
// cookie-mode browser navigation to /authorize would. The consent screen now
// requires a readable session; without it /authorize delegates to login.
func (s *OAuthSuite) session(uid string) http.Header {
	tok, err := user.JWT(jwt.MapClaims{"uid": uid}, oauthSecret)
	s.Require().NoError(err)

	h := make(http.Header)
	h.Set("Cookie", buildconf.SessionCookieName+"="+tok)

	return h
}

func pkce(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))

	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (s *OAuthSuite) authzQuery(challenge string) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", "chatgpt")
	q.Set("redirect_uri", oauthRedirect)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", "xyz")

	return q.Encode()
}

func (s *OAuthSuite) json(res *shttp.Response) map[string]any {
	m, ok := res.Data.(map[string]any)
	s.Require().True(ok, "expected map response data")

	return m
}

// registerReq builds a POST /_stormkit/oauth/register with a JSON body.
func (s *OAuthSuite) registerReq(host *hosting.Host, bodyJSON string) *hosting.RequestContext {
	h := make(http.Header)
	h.Set("Content-Type", "application/json")

	return s.req(host, http.MethodPost, "/_stormkit/oauth/register", "", h, strings.NewReader(bodyJSON))
}

// registerClient registers a client and returns the issued client_id.
func (s *OAuthSuite) registerClient(name string, redirectURIs ...string) string {
	uris := make([]string, len(redirectURIs))
	for i, u := range redirectURIs {
		uris[i] = `"` + u + `"`
	}

	body := `{"client_name":"` + name + `","redirect_uris":[` + strings.Join(uris, ",") + `]}`

	res := hosting.HandleOAuthRegister(s.registerReq(s.host(true), body))
	s.Require().Equal(http.StatusCreated, res.Status)

	id, ok := s.json(res)["client_id"].(string)
	s.Require().True(ok)
	s.Require().NotEmpty(id)

	return id
}

func (s *OAuthSuite) Test_MetadataAS() {
	res := hosting.HandleOAuthMetadataAS(s.req(s.host(true), http.MethodGet, "/.well-known/oauth-authorization-server", "", nil, nil))

	s.Equal(http.StatusOK, res.Status)
	d := s.json(res)
	s.Equal("https://app.example.com", d["issuer"])
	s.Equal("https://app.example.com/_stormkit/oauth/authorize", d["authorization_endpoint"])
	s.Equal("https://app.example.com/_stormkit/oauth/token", d["token_endpoint"])
	s.Equal("https://app.example.com/_stormkit/oauth/register", d["registration_endpoint"])
	s.Equal([]string{"S256"}, d["code_challenge_methods_supported"])
	s.Contains(d["scopes_supported"], "email")
}

func (s *OAuthSuite) Test_MetadataResource() {
	res := hosting.HandleOAuthMetadataResource(s.req(s.host(true), http.MethodGet, "/.well-known/oauth-protected-resource", "", nil, nil))

	s.Equal(http.StatusOK, res.Status)
	d := s.json(res)
	s.Equal("https://app.example.com", d["resource"])
	s.Equal([]string{"https://app.example.com"}, d["authorization_servers"])
}

func (s *OAuthSuite) Test_Authorize_RendersConsent() {
	res := hosting.HandleOAuthAuthorize(s.req(s.host(true), http.MethodGet, "/_stormkit/oauth/authorize", s.authzQuery(pkce("verifier")), s.session("user-1"), nil))

	s.Equal(http.StatusOK, res.Status)
	s.Contains(res.Headers.Get("Content-Type"), "text/html")
	s.Contains(string(res.Data.([]byte)), "Authorize access")
}

func (s *OAuthSuite) Test_Authorize_RejectsBadRedirect() {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("redirect_uri", "https://evil.example.com/cb")
	q.Set("code_challenge", pkce("v"))
	q.Set("code_challenge_method", "S256")

	res := hosting.HandleOAuthAuthorize(s.req(s.host(true), http.MethodGet, "/_stormkit/oauth/authorize", q.Encode(), nil, nil))

	s.Equal(http.StatusBadRequest, res.Status)
	s.Nil(res.Redirect, "must not redirect to an unvalidated redirect_uri")
}

// Test_Authorize_SetsAntiFramingHeaders guards the consent page against
// clickjacking: it must forbid being framed.
func (s *OAuthSuite) Test_Authorize_SetsAntiFramingHeaders() {
	res := hosting.HandleOAuthAuthorize(s.req(s.host(true), http.MethodGet, "/_stormkit/oauth/authorize", s.authzQuery(pkce("verifier")), s.session("user-1"), nil))

	s.Equal(http.StatusOK, res.Status)
	s.Equal("DENY", res.Headers.Get("X-Frame-Options"))
	s.Contains(res.Headers.Get("Content-Security-Policy"), "frame-ancestors 'none'")
}

// Test_Authorize_AcceptsMixedCaseRedirectHost verifies the redirect_uri host is
// matched case-insensitively against the preset origins.
func (s *OAuthSuite) Test_Authorize_AcceptsMixedCaseRedirectHost() {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", "chatgpt")
	q.Set("redirect_uri", "https://ChatGPT.com/connector/oauth/cb")
	q.Set("code_challenge", pkce("verifier"))
	q.Set("code_challenge_method", "S256")

	res := hosting.HandleOAuthAuthorize(s.req(s.host(true), http.MethodGet, "/_stormkit/oauth/authorize", q.Encode(), s.session("user-1"), nil))

	s.Equal(http.StatusOK, res.Status)
	s.Contains(string(res.Data.([]byte)), "Authorize access")
}

func (s *OAuthSuite) Test_Grant_RequiresAuth() {
	res := hosting.HandleOAuthGrant(s.req(s.host(true), http.MethodPost, "/_stormkit/oauth/authorize", s.authzQuery(pkce("v")), nil, nil))

	s.Equal(http.StatusUnauthorized, res.Status)
	s.Equal("access_denied", s.json(res)["error"])
}

// hostLogin returns an OAuth-enabled host with a configured delegation login URL.
func (s *OAuthSuite) hostLogin(loginURL string) *hosting.Host {
	host := s.host(true)
	host.Config.SKAuth.LoginURL = loginURL

	return host
}

// Test_Authorize_NoSession_DelegatesToLogin verifies that, without a readable
// session, /authorize redirects to the app's login URL with a return_to that
// points back at this exact request (carrying the loop-guard marker).
func (s *OAuthSuite) Test_Authorize_NoSession_DelegatesToLogin() {
	res := hosting.HandleOAuthAuthorize(s.req(s.hostLogin("/login"), http.MethodGet, "/_stormkit/oauth/authorize", s.authzQuery(pkce("verifier")), nil, nil))

	s.Equal(http.StatusFound, res.Status)
	s.Require().NotNil(res.Redirect)

	u, err := url.Parse(*res.Redirect)
	s.Require().NoError(err)
	s.Equal("https://app.example.com/login", u.Scheme+"://"+u.Host+u.Path)

	returnTo := u.Query().Get("return_to")
	s.Require().NotEmpty(returnTo)

	ret, err := url.Parse(returnTo)
	s.Require().NoError(err)
	s.Equal("/_stormkit/oauth/authorize", ret.Path)
	s.Equal("1", ret.Query().Get("sk_auth_retried"))
	s.Equal("chatgpt", ret.Query().Get("client_id"))
}

// Test_Authorize_NoSession_NoLoginURL_Errors surfaces an actionable error rather
// than dead-ending when no session exists and no login URL is configured.
func (s *OAuthSuite) Test_Authorize_NoSession_NoLoginURL_Errors() {
	res := hosting.HandleOAuthAuthorize(s.req(s.host(true), http.MethodGet, "/_stormkit/oauth/authorize", s.authzQuery(pkce("verifier")), nil, nil))

	s.Equal(http.StatusBadRequest, res.Status)
	s.Contains(string(res.Data.([]byte)), "not signed in")
}

// Test_Authorize_NoSession_RetryStops breaks the delegation loop: a return from
// login that still has no session must error, not bounce to login again.
func (s *OAuthSuite) Test_Authorize_NoSession_RetryStops() {
	q := s.authzQuery(pkce("verifier")) + "&sk_auth_retried=1"

	res := hosting.HandleOAuthAuthorize(s.req(s.hostLogin("/login"), http.MethodGet, "/_stormkit/oauth/authorize", q, nil, nil))

	s.Equal(http.StatusBadRequest, res.Status)
	s.Nil(res.Redirect)
	s.Contains(string(res.Data.([]byte)), "not signed in")
}

// Test_Grant_FromCookie mints an authorization code when the session rides a
// cookie rather than the Authorization header (the cookie-mode consent POST).
func (s *OAuthSuite) Test_Grant_FromCookie() {
	res := hosting.HandleOAuthGrant(s.req(s.host(true), http.MethodPost, "/_stormkit/oauth/authorize", s.authzQuery(pkce("verifier")), s.session("user-cookie"), nil))

	s.Require().Equal(http.StatusOK, res.Status)

	redirect, ok := s.json(res)["redirect"].(string)
	s.Require().True(ok)

	u, err := url.Parse(redirect)
	s.Require().NoError(err)
	s.NotEmpty(u.Query().Get("code"))
}

// Test_Grant_RejectsCrossOriginPost guards the cookie-authenticated grant against
// a cross-site POST: even with a valid session cookie, a mismatched Origin is
// refused so the code mint can't be driven from another site if the cookie's
// SameSite policy is ever loosened.
func (s *OAuthSuite) Test_Grant_RejectsCrossOriginPost() {
	h := s.session("user-cookie")
	h.Set("Origin", "https://evil.example.com")

	res := hosting.HandleOAuthGrant(s.req(s.host(true), http.MethodPost, "/_stormkit/oauth/authorize", s.authzQuery(pkce("verifier")), h, nil))

	s.Equal(http.StatusForbidden, res.Status)
	s.Equal("access_denied", s.json(res)["error"])
}

// Test_Grant_AcceptsSameOriginPost confirms the Origin check lets the real
// same-origin consent POST through.
func (s *OAuthSuite) Test_Grant_AcceptsSameOriginPost() {
	h := s.session("user-cookie")
	h.Set("Origin", "https://app.example.com")

	res := hosting.HandleOAuthGrant(s.req(s.host(true), http.MethodPost, "/_stormkit/oauth/authorize", s.authzQuery(pkce("verifier")), h, nil))

	s.Require().Equal(http.StatusOK, res.Status)
	s.NotEmpty(s.json(res)["redirect"])
}

// codeFromGrant runs a full authorize grant and returns the issued code.
func (s *OAuthSuite) codeFromGrant(uid, challenge string) string {
	h := make(http.Header)
	h.Set("Authorization", s.bearer(uid))

	res := hosting.HandleOAuthGrant(s.req(s.host(true), http.MethodPost, "/_stormkit/oauth/authorize", s.authzQuery(challenge), h, nil))
	s.Require().Equal(http.StatusOK, res.Status)

	redirect, ok := s.json(res)["redirect"].(string)
	s.Require().True(ok)

	u, err := url.Parse(redirect)
	s.Require().NoError(err)
	s.Equal("xyz", u.Query().Get("state"))

	code := u.Query().Get("code")
	s.Require().NotEmpty(code)

	return code
}

func (s *OAuthSuite) tokenReq(form url.Values) *hosting.RequestContext {
	h := make(http.Header)
	h.Set("Content-Type", "application/x-www-form-urlencoded")

	return s.req(s.host(true), http.MethodPost, "/_stormkit/oauth/token", "", h, strings.NewReader(form.Encode()))
}

func (s *OAuthSuite) Test_Token_ExchangeSucceeds() {
	verifier := "the-code-verifier-value-1234567890"
	code := s.codeFromGrant("user-uuid-42", pkce(verifier))

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", "chatgpt")
	form.Set("redirect_uri", oauthRedirect)
	form.Set("code_verifier", verifier)

	res := hosting.HandleOAuthToken(s.tokenReq(form))

	s.Equal(http.StatusOK, res.Status)
	d := s.json(res)
	s.Equal("Bearer", d["token_type"])
	s.Equal(600, d["expires_in"])

	accessToken, ok := d["access_token"].(string)
	s.Require().True(ok)

	// The access token is in the SkAuth format, so the edge validation accepts
	// it: it must carry the resource-owner uid and this app's audience.
	claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: accessToken, Secret: oauthSecret, MaxMins: 10})
	s.Require().NotNil(claims)
	s.Equal("user-uuid-42", claims["uid"])
	s.Equal("https://app.example.com", claims["aud"])
}

func (s *OAuthSuite) Test_Token_RejectsWrongVerifier() {
	code := s.codeFromGrant("user-uuid-42", pkce("real-verifier-value-000000000000"))

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", "chatgpt")
	form.Set("redirect_uri", oauthRedirect)
	form.Set("code_verifier", "wrong-verifier-value-9999999999999")

	res := hosting.HandleOAuthToken(s.tokenReq(form))

	s.Equal(http.StatusBadRequest, res.Status)
	s.Equal("invalid_grant", s.json(res)["error"])
}

func (s *OAuthSuite) Test_Token_RejectsWrongClientID() {
	verifier := "client-bind-verifier-1234567890abc"
	code := s.codeFromGrant("user-uuid-42", pkce(verifier))

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", "not-chatgpt")
	form.Set("redirect_uri", oauthRedirect)
	form.Set("code_verifier", verifier)

	res := hosting.HandleOAuthToken(s.tokenReq(form))

	s.Equal(http.StatusBadRequest, res.Status)
	s.Equal("invalid_grant", s.json(res)["error"])
}

func (s *OAuthSuite) Test_Token_CodeIsSingleUse() {
	verifier := "single-use-verifier-1234567890abcd"
	code := s.codeFromGrant("user-uuid-7", pkce(verifier))

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", "chatgpt")
	form.Set("redirect_uri", oauthRedirect)
	form.Set("code_verifier", verifier)

	first := hosting.HandleOAuthToken(s.tokenReq(form))
	s.Require().Equal(http.StatusOK, first.Status)

	second := hosting.HandleOAuthToken(s.tokenReq(form))
	s.Equal(http.StatusBadRequest, second.Status)
	s.Equal("invalid_grant", s.json(second)["error"])
}

// --- Dynamic Client Registration (RFC 7591) ---

func (s *OAuthSuite) Test_Register_Success() {
	body := `{"client_name":"ChatGPT","redirect_uris":["https://chatgpt.com/connector/oauth/cb"]}`

	res := hosting.HandleOAuthRegister(s.registerReq(s.host(true), body))

	s.Equal(http.StatusCreated, res.Status)
	d := s.json(res)
	s.NotEmpty(d["client_id"])
	s.Equal("none", d["token_endpoint_auth_method"])
	s.Equal("ChatGPT", d["client_name"])
	// Public PKCE client: no secret must ever be issued.
	_, hasSecret := d["client_secret"]
	s.False(hasSecret, "public clients must not receive a client_secret")
}

func (s *OAuthSuite) Test_Register_RejectsRedirectOffPreset() {
	body := `{"redirect_uris":["https://evil.example.com/cb"]}`

	res := hosting.HandleOAuthRegister(s.registerReq(s.host(true), body))

	s.Equal(http.StatusBadRequest, res.Status)
	s.Equal("invalid_redirect_uri", s.json(res)["error"])
}

func (s *OAuthSuite) Test_Register_RequiresRedirectURIs() {
	res := hosting.HandleOAuthRegister(s.registerReq(s.host(true), `{"client_name":"NoRedirect"}`))

	s.Equal(http.StatusBadRequest, res.Status)
	s.Equal("invalid_redirect_uri", s.json(res)["error"])
}

func (s *OAuthSuite) Test_Register_RejectsConfidentialAuthMethod() {
	body := `{"redirect_uris":["https://chatgpt.com/connector/oauth/cb"],"token_endpoint_auth_method":"client_secret_basic"}`

	res := hosting.HandleOAuthRegister(s.registerReq(s.host(true), body))

	s.Equal(http.StatusBadRequest, res.Status)
	s.Equal("invalid_client_metadata", s.json(res)["error"])
}

func (s *OAuthSuite) Test_Register_RejectsOversizedBody() {
	// An oversized body must be refused before anything is written to Redis, so
	// the unauthenticated endpoint can't be used to amplify storage.
	body := `{"client_name":"` + strings.Repeat("a", 32*1024) + `","redirect_uris":["https://chatgpt.com/connector/oauth/cb"]}`

	res := hosting.HandleOAuthRegister(s.registerReq(s.host(true), body))

	s.Equal(http.StatusBadRequest, res.Status)
	s.Equal("invalid_client_metadata", s.json(res)["error"])
}

func (s *OAuthSuite) Test_Register_RejectsTooManyRedirectURIs() {
	uris := make([]string, 0, 32)
	for i := 0; i < 32; i++ {
		uris = append(uris, `"https://chatgpt.com/connector/oauth/cb"`)
	}

	body := `{"redirect_uris":[` + strings.Join(uris, ",") + `]}`

	res := hosting.HandleOAuthRegister(s.registerReq(s.host(true), body))

	s.Equal(http.StatusBadRequest, res.Status)
	s.Equal("invalid_redirect_uri", s.json(res)["error"])
}

func (s *OAuthSuite) Test_Register_Disabled_FallsThrough() {
	// When OAuth is off the path is not reserved: serveOAuth forwards to normal
	// app serving instead of registering a client.
	res := hosting.HandleOAuthRegister(s.registerReq(s.host(false), `{"redirect_uris":["https://chatgpt.com/connector/oauth/cb"]}`))

	s.NotEqual(http.StatusCreated, res.Status)
}

func (s *OAuthSuite) Test_Register_RateLimited() {
	host := &hosting.Host{Name: "ratelimit.example.com", Config: s.host(true).Config}
	hosting.ResetOAuthRateLimit(host.Name)

	body := `{"redirect_uris":["https://chatgpt.com/connector/oauth/cb"]}`

	for i := 0; i < hosting.OAuthRegisterRateMax(); i++ {
		res := hosting.HandleOAuthRegister(s.registerReq(host, body))
		s.Require().Equal(http.StatusCreated, res.Status, "registration %d should succeed", i)
	}

	res := hosting.HandleOAuthRegister(s.registerReq(host, body))
	s.Equal(http.StatusTooManyRequests, res.Status)
}

// Test_Authorize_RegisteredClient_BindsRedirect verifies a registered client is
// held to its declared redirect_uris, even for another path on the same preset
// origin (which the origin-only gate alone would permit).
func (s *OAuthSuite) Test_Authorize_RegisteredClient_BindsRedirect() {
	clientID := s.registerClient("ChatGPT", oauthRedirect)

	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", "https://chatgpt.com/connector/oauth/other")
	q.Set("code_challenge", pkce("verifier"))
	q.Set("code_challenge_method", "S256")

	res := hosting.HandleOAuthAuthorize(s.req(s.host(true), http.MethodGet, "/_stormkit/oauth/authorize", q.Encode(), nil, nil))

	s.Equal(http.StatusBadRequest, res.Status)
	s.Contains(string(res.Data.([]byte)), "not registered")
}

// Test_Authorize_RegisteredClient_ShowsName renders the client's declared
// client_name on the consent screen rather than the opaque client_id.
func (s *OAuthSuite) Test_Authorize_RegisteredClient_ShowsName() {
	clientID := s.registerClient("Claude", oauthRedirect)

	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", oauthRedirect)
	q.Set("code_challenge", pkce("verifier"))
	q.Set("code_challenge_method", "S256")

	res := hosting.HandleOAuthAuthorize(s.req(s.host(true), http.MethodGet, "/_stormkit/oauth/authorize", q.Encode(), s.session("user-1"), nil))

	s.Equal(http.StatusOK, res.Status)
	s.Contains(string(res.Data.([]byte)), "Claude")
}

// Test_Authorize_ShowsScopeLabels renders human-readable labels for known
// scopes on the consent screen.
func (s *OAuthSuite) Test_Authorize_ShowsScopeLabels() {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", "chatgpt")
	q.Set("redirect_uri", oauthRedirect)
	q.Set("code_challenge", pkce("verifier"))
	q.Set("code_challenge_method", "S256")
	q.Set("scope", "email profile")

	res := hosting.HandleOAuthAuthorize(s.req(s.host(true), http.MethodGet, "/_stormkit/oauth/authorize", q.Encode(), s.session("user-1"), nil))

	s.Equal(http.StatusOK, res.Status)
	html := string(res.Data.([]byte))
	s.Contains(html, "Access your email address")
	s.Contains(html, "Access your basic profile information")
}
