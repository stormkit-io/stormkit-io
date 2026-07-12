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
const oauthRedirect = "https://client.example.com/callback"

type OAuthSuite struct {
	suite.Suite
}

func TestOAuthSuite(t *testing.T) {
	suite.Run(t, new(OAuthSuite))
}

func (s *OAuthSuite) host(enabled bool) *hosting.Host {
	return &hosting.Host{
		Name: "app.example.com",
		Config: &appconf.Config{
			SKAuth: &buildconf.SKAuthConf{
				Secret:         oauthSecret,
				Status:         true,
				TTL:            10,
				AllowedOrigins: []string{"https://client.example.com"},
				OAuthServer:    &buildconf.OAuthServerConf{Enabled: enabled},
			},
		},
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

func (s *OAuthSuite) Test_MetadataAS() {
	res := hosting.HandleOAuthMetadataAS(s.req(s.host(true), http.MethodGet, "/.well-known/oauth-authorization-server", "", nil, nil))

	s.Equal(http.StatusOK, res.Status)
	d := s.json(res)
	s.Equal("https://app.example.com", d["issuer"])
	s.Equal("https://app.example.com/_stormkit/oauth/authorize", d["authorization_endpoint"])
	s.Equal("https://app.example.com/_stormkit/oauth/token", d["token_endpoint"])
	s.Equal([]string{"S256"}, d["code_challenge_methods_supported"])
}

func (s *OAuthSuite) Test_MetadataResource() {
	res := hosting.HandleOAuthMetadataResource(s.req(s.host(true), http.MethodGet, "/.well-known/oauth-protected-resource", "", nil, nil))

	s.Equal(http.StatusOK, res.Status)
	d := s.json(res)
	s.Equal("https://app.example.com", d["resource"])
	s.Equal([]string{"https://app.example.com"}, d["authorization_servers"])
}

func (s *OAuthSuite) Test_Authorize_RendersConsent() {
	res := hosting.HandleOAuthAuthorize(s.req(s.host(true), http.MethodGet, "/_stormkit/oauth/authorize", s.authzQuery(pkce("verifier")), nil, nil))

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

func (s *OAuthSuite) Test_Grant_RequiresAuth() {
	res := hosting.HandleOAuthGrant(s.req(s.host(true), http.MethodPost, "/_stormkit/oauth/authorize", s.authzQuery(pkce("v")), nil, nil))

	s.Equal(http.StatusUnauthorized, res.Status)
	s.Equal("access_denied", s.json(res)["error"])
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
	form.Set("redirect_uri", oauthRedirect)
	form.Set("code_verifier", "wrong-verifier-value-9999999999999")

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
	form.Set("redirect_uri", oauthRedirect)
	form.Set("code_verifier", verifier)

	first := hosting.HandleOAuthToken(s.tokenReq(form))
	s.Require().Equal(http.StatusOK, first.Status)

	second := hosting.HandleOAuthToken(s.tokenReq(form))
	s.Equal(http.StatusBadRequest, second.Status)
	s.Equal("invalid_grant", s.json(second)["error"])
}
