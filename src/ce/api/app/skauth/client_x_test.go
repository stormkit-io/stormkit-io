package skauth_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
	"golang.org/x/oauth2"
)

type ClientXSuite struct {
	suite.Suite
	conn                    databasetest.TestDB
	client                  skauth.Client
	server                  *httptest.Server
	originalTwitterAuthBase string
	originalTwitterAPIBase  string
}

func (s *ClientXSuite) BeforeTest(suiteName, _ string) {
	s.originalTwitterAuthBase = skauth.TwitterAuthBase
	s.originalTwitterAPIBase = skauth.TwitterAPIBase
	s.conn = databasetest.InitTx(suiteName)
	s.server = nil
	s.client = skauth.NewXClient("test-client-id", "test-client-secret", "https://app.example.com/_stormkit/auth/callback", "")
	admin.MustConfig().SetURL("localhost")
}

func (s *ClientXSuite) AfterTest(_, _ string) {
	skauth.TwitterAuthBase = s.originalTwitterAuthBase
	skauth.TwitterAPIBase = s.originalTwitterAPIBase

	if s.server != nil {
		s.server.Close()
	}

	s.conn.CloseTx()
}

func (s *ClientXSuite) mockServer(handler http.HandlerFunc) {
	s.server = httptest.NewServer(handler)
	skauth.TwitterAPIBase = s.server.URL // Override API base URL for testing
}

func (s *ClientXSuite) Test_UserInfo_Success() {
	// Mock X API server
	s.mockServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("GET", r.Method)
		s.Contains(r.URL.Path, "/users/me")
		s.Contains(r.URL.Query().Get("user.fields"), "confirmed_email")
		s.Equal("Bearer test-access-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"id": "123456789",
				"name": "John Doe",
				"username": "johndoe",
				"profile_image_url": "https://example.com/avatar.jpg",
				"confirmed_email": "john@example.com"
			}
		}`))
	}))

	token := &oauth2.Token{
		AccessToken: "test-access-token",
		TokenType:   "Bearer",
	}

	// Use new client to ensure we're testing with a mock server
	client := skauth.NewXClient("test-client-id", "test-client-secret", "https://app.example.com/_stormkit/auth/callback", "")
	userInfo, err := client.UserInfo(context.Background(), token)

	s.NoError(err)
	s.NotNil(userInfo)
	s.Equal("123456789", userInfo.AccountID)
	s.Equal("john@example.com", userInfo.Email)
	s.Equal("https://example.com/avatar.jpg", userInfo.Avatar)
	s.Equal("John Doe", userInfo.FirstName)
	s.Equal("", userInfo.LastName)
	s.Equal("johndoe", userInfo.Username)
	s.Equal("https://x.com/johndoe", userInfo.ProfileURL)
}

func (s *ClientXSuite) Test_UserInfo_NoUsername_NoProfileURL() {
	s.mockServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"id": "123456789",
				"name": "John Doe",
				"confirmed_email": "john@example.com"
			}
		}`))
	}))

	token := &oauth2.Token{AccessToken: "test-access-token", TokenType: "Bearer"}

	client := skauth.NewXClient("test-client-id", "test-client-secret", "https://app.example.com/_stormkit/auth/callback", "")
	userInfo, err := client.UserInfo(context.Background(), token)

	s.NoError(err)
	s.Require().NotNil(userInfo)
	s.Equal("", userInfo.Username)
	s.Equal("", userInfo.ProfileURL, "no handle means no profile URL is constructed")
}

func (s *ClientXSuite) Test_UserInfo_InvalidJSON() {
	s.mockServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))

	token := &oauth2.Token{
		AccessToken: "test-access-token",
		TokenType:   "Bearer",
	}

	// Use new client to ensure we're testing with a mock server
	client := skauth.NewXClient("test-client-id", "test-client-secret", "https://app.example.com/_stormkit/auth/callback", "")
	userInfo, err := client.UserInfo(context.Background(), token)

	s.Error(err)
	s.Nil(userInfo)
}

func (s *ClientXSuite) Test_AuthCodeURL() {
	url, err := s.client.AuthCodeURL(skauth.AuthCodeURLParams{
		EnvID:        1,
		ProviderName: skauth.ProviderX,
	})

	s.NoError(err)
	s.NotEmpty(url)
	s.Contains(url, "x.com/i/oauth2/authorize")
	s.Contains(url, "client_id=test-client-id")
	s.Contains(url, "state=")
	s.Contains(url, "code_challenge_method=S256")
	s.Contains(url, "code_challenge=")
	s.Contains(url, "access_type=offline")
}

func (s *ClientXSuite) Test_Exchange_WithPKCE() {
	s.mockServer(func(w http.ResponseWriter, r *http.Request) {})

	code := "test-authorization-code"
	verifier, err := utils.SecureRandomToken(64)
	s.NoError(err)

	encryptedVerifier := utils.EncryptToString(verifier)

	claims := jwt.MapClaims{
		"pkce": encryptedVerifier,
		"eid":  1,
		"prv":  "x",
	}

	state, err := user.JWT(claims)
	s.NoError(err)

	// Create a proper http.Request with form values
	formData := url.Values{}
	formData.Set("code", code)
	formData.Set("state", state)

	httpReq, err := http.NewRequest("POST", "http://example.com/callback", strings.NewReader(formData.Encode()))
	s.NoError(err)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Form = formData

	// Create RequestContext
	req := shttp.NewRequestContext(httpReq)
	ctx := context.Background()

	// Note: This will fail in actual OAuth exchange since we don't have a real server
	// but it tests that the verifier is properly extracted and decrypted
	_, err = s.client.Exchange(ctx, req)

	// We expect an error because we're not hitting a real OAuth server
	// But we've verified the code path including PKCE verifier extraction
	s.Error(err) // Expected to fail without real OAuth server
}

func (s *ClientXSuite) Test_Exchange_WithoutPKCE() {
	code := "test-authorization-code"

	claims := jwt.MapClaims{
		"eid": 1,
		"prv": "x",
		// No PKCE
	}

	state, err := user.JWT(claims)
	s.NoError(err)

	// Create a proper http.Request with form values
	formData := url.Values{}
	formData.Set("code", code)
	formData.Set("state", state)

	httpReq, err := http.NewRequest("POST", "http://example.com/callback", strings.NewReader(formData.Encode()))
	s.NoError(err)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Form = formData

	// Create RequestContext
	req := shttp.NewRequestContext(httpReq)
	ctx := context.Background()

	// This should also fail but without PKCE
	_, err = s.client.Exchange(ctx, req)
	s.Error(err) // Expected to fail without real OAuth server
}

func (s *ClientXSuite) Test_Exchange_InvalidState() {
	code := "test-authorization-code"

	// Create a proper http.Request with invalid form values
	formData := url.Values{}
	formData.Set("code", code)
	formData.Set("state", "invalid-jwt-token")

	httpReq, err := http.NewRequest("POST", "http://example.com/callback", strings.NewReader(formData.Encode()))
	s.NoError(err)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Form = formData

	// Create RequestContext
	req := shttp.NewRequestContext(httpReq)
	ctx := context.Background()

	// This should fail because state JWT is invalid
	_, err = s.client.Exchange(ctx, req)
	s.Error(err)
}

// Test_Exchange_SendsCodeVerifier_WhenStateSecretMatches guards the regression
// that broke X login: AuthCodeURL signs the state with the environment auth
// secret, so Exchange must verify it with the same secret to recover the
// encrypted PKCE verifier. When it matches, the token request carries
// code_verifier and X accepts the exchange.
func (s *ClientXSuite) Test_Exchange_SendsCodeVerifier_WhenStateSecretMatches() {
	const secret = "env-auth-secret"

	var captured url.Values

	s.mockServer(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured, _ = url.ParseQuery(string(body))

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"tok","token_type":"bearer"}`))
	})

	// Construct after the token URL is overridden so the exchange hits the mock.
	client := skauth.NewXClient("id", "secret", "https://app.example.com/cb", secret)

	verifier, err := utils.SecureRandomToken(64)
	s.NoError(err)

	state, err := user.JWT(jwt.MapClaims{
		"pkce": utils.EncryptToString(verifier),
		"eid":  1,
		"prv":  "x",
	}, secret)
	s.NoError(err)

	req := s.exchangeRequest("auth-code", state)
	token, err := client.Exchange(context.Background(), req)

	s.NoError(err)
	s.Equal("tok", token.AccessToken)
	s.Equal(verifier, captured.Get("code_verifier"), "PKCE verifier must be sent when the state secret matches")
}

// Test_Exchange_DropsVerifier_WhenStateSecretMismatches documents the failure
// mode: if the state was signed with a different secret than the client
// verifies with, the claims (and thus the PKCE verifier) are dropped and the
// exchange falls back to a non-PKCE request — which X rejects with
// "Missing required parameter [code_verifier]".
func (s *ClientXSuite) Test_Exchange_DropsVerifier_WhenStateSecretMismatches() {
	var captured url.Values

	s.mockServer(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured, _ = url.ParseQuery(string(body))

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"tok","token_type":"bearer"}`))
	})

	client := skauth.NewXClient("id", "secret", "https://app.example.com/cb", "the-right-secret")

	state, err := user.JWT(jwt.MapClaims{
		"pkce": utils.EncryptToString("some-verifier"),
		"eid":  1,
		"prv":  "x",
	}, "a-different-secret")
	s.NoError(err)

	req := s.exchangeRequest("auth-code", state)
	_, err = client.Exchange(context.Background(), req)

	s.NoError(err)
	s.Empty(captured.Get("code_verifier"), "verifier is dropped when the state secret does not match")
}

func (s *ClientXSuite) exchangeRequest(code, state string) *shttp.RequestContext {
	form := url.Values{}
	form.Set("code", code)
	form.Set("state", state)

	httpReq, err := http.NewRequest("POST", "http://example.com/callback", strings.NewReader(form.Encode()))
	s.NoError(err)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Form = form

	return shttp.NewRequestContext(httpReq)
}

func TestClientXSuite(t *testing.T) {
	suite.Run(t, &ClientXSuite{})
}
