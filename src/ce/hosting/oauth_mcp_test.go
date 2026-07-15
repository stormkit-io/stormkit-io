package hosting_test

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	hosting "github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// hostWith builds a host with the given OAuth server configuration on top of the
// shared test secret, so MCP-specific tests can opt into a resource path or
// loopback support without touching the base host() helper.
func (s *OAuthSuite) hostWith(conf buildconf.OAuthServerConf) *hosting.Host {
	conf.Enabled = true

	return &hosting.Host{
		Name: "app.example.com",
		Config: &appconf.Config{
			SKAuth: &buildconf.SKAuthConf{
				Secret:      oauthSecret,
				Status:      true,
				TTL:         10,
				OAuthServer: &conf,
			},
		},
	}
}

// jsonMaybe returns the response data as a map, or nil when it is not one (e.g.
// a fall-through response from HandlerForward).
func (s *OAuthSuite) jsonMaybe(res *shttp.Response) map[string]any {
	m, _ := res.Data.(map[string]any)

	return m
}

// --- P0: path-qualified resource + RFC 9728 path-aware well-known ---

func (s *OAuthSuite) Test_MetadataResource_IncludesConfiguredPath() {
	host := s.hostWith(buildconf.OAuthServerConf{ResourcePath: "/mcp"})

	res := hosting.HandleOAuthMetadataResource(s.req(host, http.MethodGet, "/.well-known/oauth-protected-resource", "", nil, nil))

	s.Equal(http.StatusOK, res.Status)
	d := s.json(res)
	s.Equal("https://app.example.com/mcp", d["resource"])
	s.Equal([]string{"https://app.example.com"}, d["authorization_servers"])
}

func (s *OAuthSuite) Test_MetadataResource_PathVariant_Match() {
	host := s.hostWith(buildconf.OAuthServerConf{ResourcePath: "/mcp"})

	res := hosting.HandleOAuthMetadataResource(s.req(host, http.MethodGet, "/.well-known/oauth-protected-resource/mcp", "", nil, nil))

	s.Equal(http.StatusOK, res.Status)
	s.Equal("https://app.example.com/mcp", s.json(res)["resource"])
}

// A probe for a path this environment does not serve must not answer with the
// metadata document; it falls through to normal app serving.
func (s *OAuthSuite) Test_MetadataResource_PathVariant_Mismatch_FallsThrough() {
	host := s.hostWith(buildconf.OAuthServerConf{ResourcePath: "/mcp"})

	res := hosting.HandleOAuthMetadataResource(s.req(host, http.MethodGet, "/.well-known/oauth-protected-resource/other", "", nil, nil))

	s.Nil(s.jsonMaybe(res)["resource"], "mismatched path must not return the resource metadata")
}

// --- CORS on discovery documents ---

func (s *OAuthSuite) Test_MetadataResource_HasCORS() {
	res := hosting.HandleOAuthMetadataResource(s.req(s.host(true), http.MethodGet, "/.well-known/oauth-protected-resource", "", nil, nil))

	s.Equal("*", res.Headers.Get("Access-Control-Allow-Origin"))
}

func (s *OAuthSuite) Test_MetadataAS_AdvertisesRefreshAndOfflineAccess() {
	res := hosting.HandleOAuthMetadataAS(s.req(s.host(true), http.MethodGet, "/.well-known/oauth-authorization-server", "", nil, nil))

	d := s.json(res)
	s.Contains(d["grant_types_supported"], "refresh_token")
	s.Contains(d["scopes_supported"], "offline_access")
}

// --- P1: loopback redirects (RFC 8252) ---

func (s *OAuthSuite) loopbackQuery(redirect string) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", "claude-code")
	q.Set("redirect_uri", redirect)
	q.Set("code_challenge", pkce("verifier"))
	q.Set("code_challenge_method", "S256")

	return q.Encode()
}

func (s *OAuthSuite) Test_Authorize_Loopback_AllowedWhenEnabled() {
	host := s.hostWith(buildconf.OAuthServerConf{AllowLoopback: true})

	res := hosting.HandleOAuthAuthorize(s.req(host, http.MethodGet, "/_stormkit/oauth/authorize", s.loopbackQuery("http://127.0.0.1:54321/callback"), s.session("user-1"), nil))

	s.Equal(http.StatusOK, res.Status)
	s.Contains(string(res.Data.([]byte)), "Authorize access")
}

func (s *OAuthSuite) Test_Authorize_Loopback_RejectedWhenDisabled() {
	res := hosting.HandleOAuthAuthorize(s.req(s.host(true), http.MethodGet, "/_stormkit/oauth/authorize", s.loopbackQuery("http://127.0.0.1:54321/callback"), nil, nil))

	s.Equal(http.StatusBadRequest, res.Status)
}

// A registered native client declares a bare loopback redirect but returns on an
// ephemeral port; the port must be ignored (RFC 8252 §7.3).
func (s *OAuthSuite) Test_Authorize_Loopback_RegisteredClient_IgnoresPort() {
	host := s.hostWith(buildconf.OAuthServerConf{AllowLoopback: true})

	body := `{"client_name":"Claude Code","redirect_uris":["http://127.0.0.1/callback"]}`
	reg := hosting.HandleOAuthRegister(s.registerReq(host, body))
	s.Require().Equal(http.StatusCreated, reg.Status)

	clientID := s.json(reg)["client_id"].(string)

	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", "http://127.0.0.1:61023/callback")
	q.Set("code_challenge", pkce("verifier"))
	q.Set("code_challenge_method", "S256")

	res := hosting.HandleOAuthAuthorize(s.req(host, http.MethodGet, "/_stormkit/oauth/authorize", q.Encode(), s.session("user-1"), nil))

	s.Equal(http.StatusOK, res.Status)
	s.Contains(string(res.Data.([]byte)), "Authorize access")
}

// --- P0: refresh tokens ---

// grantWithScope runs a full authorize grant carrying scope and returns the code.
func (s *OAuthSuite) grantWithScope(host *hosting.Host, uid, challenge, scope, resource string) string {
	h := make(http.Header)
	h.Set("Authorization", s.bearer(uid))

	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", "chatgpt")
	q.Set("redirect_uri", oauthRedirect)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", "xyz")
	q.Set("scope", scope)

	if resource != "" {
		q.Set("resource", resource)
	}

	res := hosting.HandleOAuthGrant(s.req(host, http.MethodPost, "/_stormkit/oauth/authorize", q.Encode(), h, nil))
	s.Require().Equal(http.StatusOK, res.Status)

	u, err := url.Parse(s.json(res)["redirect"].(string))
	s.Require().NoError(err)

	return u.Query().Get("code")
}

func (s *OAuthSuite) exchange(host *hosting.Host, code, verifier string) *shttp.Response {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", "chatgpt")
	form.Set("redirect_uri", oauthRedirect)
	form.Set("code_verifier", verifier)

	h := make(http.Header)
	h.Set("Content-Type", "application/x-www-form-urlencoded")

	return hosting.HandleOAuthToken(s.req(host, http.MethodPost, "/_stormkit/oauth/token", "", h, strings.NewReader(form.Encode())))
}

func (s *OAuthSuite) Test_Token_IssuesRefreshToken_WithOfflineAccess() {
	verifier := "offline-verifier-value-1234567890a"
	code := s.grantWithScope(s.host(true), "user-1", pkce(verifier), "openid offline_access", "")

	res := s.exchange(s.host(true), code, verifier)

	s.Equal(http.StatusOK, res.Status)
	s.NotEmpty(s.json(res)["refresh_token"], "offline_access must yield a refresh token")
}

func (s *OAuthSuite) Test_Token_NoRefreshToken_WithoutOfflineAccess() {
	verifier := "no-offline-verifier-value-12345678"
	code := s.grantWithScope(s.host(true), "user-1", pkce(verifier), "openid", "")

	res := s.exchange(s.host(true), code, verifier)

	s.Equal(http.StatusOK, res.Status)
	_, has := s.json(res)["refresh_token"]
	s.False(has, "no refresh token without offline_access")
}

// refreshReq builds a refresh_token grant request.
func (s *OAuthSuite) refreshReq(host *hosting.Host, refresh, clientID string) *shttp.Response {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refresh)

	if clientID != "" {
		form.Set("client_id", clientID)
	}

	h := make(http.Header)
	h.Set("Content-Type", "application/x-www-form-urlencoded")

	return hosting.HandleOAuthToken(s.req(host, http.MethodPost, "/_stormkit/oauth/token", "", h, strings.NewReader(form.Encode())))
}

func (s *OAuthSuite) Test_Refresh_RotatesToken() {
	verifier := "rotate-verifier-value-1234567890ab"
	code := s.grantWithScope(s.host(true), "user-42", pkce(verifier), "offline_access", "")

	first := s.exchange(s.host(true), code, verifier)
	refresh := s.json(first)["refresh_token"].(string)

	// A refresh grant mints a fresh access token and rotates the refresh token.
	res := s.refreshReq(s.host(true), refresh, "chatgpt")
	s.Equal(http.StatusOK, res.Status)

	d := s.json(res)
	s.NotEmpty(d["access_token"])

	rotated, ok := d["refresh_token"].(string)
	s.Require().True(ok)
	s.NotEqual(refresh, rotated, "refresh token must rotate")

	claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: d["access_token"].(string), Secret: oauthSecret, MaxMins: 10})
	s.Require().NotNil(claims)
	s.Equal("user-42", claims["uid"])

	// The consumed token must not be replayable.
	replay := s.refreshReq(s.host(true), refresh, "chatgpt")
	s.Equal(http.StatusBadRequest, replay.Status)
	s.Equal("invalid_grant", s.json(replay)["error"])
}

func (s *OAuthSuite) Test_Refresh_InvalidToken() {
	res := s.refreshReq(s.host(true), "not-a-real-refresh-token", "chatgpt")

	s.Equal(http.StatusBadRequest, res.Status)
	s.Equal("invalid_grant", s.json(res)["error"])
}

func (s *OAuthSuite) Test_Refresh_ClientMismatch() {
	verifier := "mismatch-verifier-value-1234567890"
	code := s.grantWithScope(s.host(true), "user-9", pkce(verifier), "offline_access", "")

	first := s.exchange(s.host(true), code, verifier)
	refresh := s.json(first)["refresh_token"].(string)

	res := s.refreshReq(s.host(true), refresh, "someone-else")
	s.Equal(http.StatusBadRequest, res.Status)
	s.Equal("invalid_grant", s.json(res)["error"])
}

// --- P1: audience binding (RFC 8707) ---

func (s *OAuthSuite) Test_Token_AudienceFromResourceIndicator() {
	host := s.hostWith(buildconf.OAuthServerConf{ResourcePath: "/mcp"})
	resource := "https://app.example.com/mcp"

	verifier := "audience-verifier-value-1234567890"
	code := s.grantWithScope(host, "user-1", pkce(verifier), "openid", resource)

	res := s.exchange(host, code, verifier)
	s.Equal(http.StatusOK, res.Status)

	claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: s.json(res)["access_token"].(string), Secret: oauthSecret, MaxMins: 10})
	s.Require().NotNil(claims)
	s.Equal(resource, claims["aud"], "aud must echo the requested resource")
}

func (s *OAuthSuite) Test_Grant_RejectsUnknownResource() {
	h := make(http.Header)
	h.Set("Authorization", s.bearer("user-1"))

	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", "chatgpt")
	q.Set("redirect_uri", oauthRedirect)
	q.Set("code_challenge", pkce("verifier"))
	q.Set("code_challenge_method", "S256")
	q.Set("resource", "https://someone-else.example.com/mcp")

	res := hosting.HandleOAuthGrant(s.req(s.host(true), http.MethodPost, "/_stormkit/oauth/authorize", q.Encode(), h, nil))

	s.Equal(http.StatusBadRequest, res.Status)
	s.Equal("invalid_target", s.json(res)["error"])
}

// Audience defaults to the resource id when no indicator is supplied and a path
// is configured.
func (s *OAuthSuite) Test_Token_AudienceDefaultsToResourceID() {
	host := s.hostWith(buildconf.OAuthServerConf{ResourcePath: "/mcp"})

	verifier := "default-aud-verifier-value-1234567"
	code := s.grantWithScope(host, "user-1", pkce(verifier), "openid", "")

	res := s.exchange(host, code, verifier)
	s.Equal(http.StatusOK, res.Status)

	claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: s.json(res)["access_token"].(string), Secret: oauthSecret, MaxMins: 10})
	s.Require().NotNil(claims)
	s.Equal("https://app.example.com/mcp", claims["aud"])
}
