package hosting_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/mock"
	"golang.org/x/oauth2"
)

// stateToken mints the OAuth2 state JWT the provider hands back on the callback.
// The hosting callback only reads "prv" and "ref" — the environment is resolved
// from the request Host, not from the state — so those are the only claims set.
func (s *WithSKAuthOAuthSuite) stateToken(provider, ref string) string {
	// Signed with the environment secret because the callback now verifies the
	// state against it (a token minted for another tenant is rejected).
	token, err := user.JWT(jwt.MapClaims{
		"prv": provider,
		"ref": ref,
	}, "test-secret-padded-to-32-chars!!")

	s.Require().NoError(err)

	return token
}

// setupCallbackEnv provisions an environment wired for the full callback path:
// auth enabled with a signing secret, a real schema store with the auth table
// created, and an enabled Google provider.
func (s *WithSKAuthOAuthSuite) setupCallbackEnv(providerStatus bool) *hosting.Host {
	return s.setupCallbackEnvWith(providerStatus, &buildconf.SKAuthConf{
		Secret:         "test-secret-padded-to-32-chars!!",
		Status:         true,
		AllowedOrigins: []string{"https://app.example.com"},
	})
}

func (s *WithSKAuthOAuthSuite) setupCallbackEnvWith(providerStatus bool, auth *buildconf.SKAuthConf) *hosting.Host {
	env := s.MockEnv(s.app, map[string]any{
		"AuthConf": auth,
		"SchemaConf": &buildconf.SchemaConf{
			SchemaName:        s.conn.Cfg.Schema,
			DBName:            s.conn.Cfg.DBName,
			Port:              s.conn.Cfg.Port,
			Host:              s.conn.Cfg.Host,
			MigrationUserName: s.conn.Cfg.User,
			MigrationPassword: s.conn.Cfg.Password,
			AppUserName:       s.conn.Cfg.User,
			AppPassword:       s.conn.Cfg.Password,
		},
	})

	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeMigrations)
	s.Require().NoError(err)
	s.Require().NoError(store.CreateAuthTable(context.Background()))

	s.Require().NoError(skauth.NewStore().SaveProvider(context.Background(), skauth.SaveProviderArgs{
		EnvID: env.ID,
		AppID: s.app.ID,
		Provider: &skauth.Provider{
			Name:   skauth.ProviderGoogle,
			Status: providerStatus,
		},
	}))

	// In production the request host config is loaded from the same env, so
	// mirror the cookie domain onto it (hostFor otherwise stubs a bare config).
	host := s.hostFor(env.ID)
	host.Config.SKAuth.CookieDomain = auth.CookieDomain

	return host
}

func (s *WithSKAuthOAuthSuite) Test_Callback_Success() {
	host := s.setupCallbackEnv(true)

	token := &oauth2.Token{}

	s.mockClient.On("Exchange", mock.Anything, mock.MatchedBy(func(req *shttp.RequestContext) bool {
		return req.FormValue("code") == "test-code"
	})).Return(token, nil).Once()

	s.mockClient.On("UserInfo", mock.Anything, token).Return(&skauth.UserInfo{
		AccountID:     "test-account-id",
		Email:         "jane@stormkit.io",
		EmailVerified: true,
		FirstName:     "Jane",
		LastName:      "Doe",
		Avatar:        "link-to-avatar",
	}, nil).Once()

	state := s.stateToken(skauth.ProviderGoogle, "https://app.example.com/login")

	res, err := hosting.ServeAuth(s.oauthRequest(
		host,
		fmt.Sprintf("/_stormkit/auth/callback?state=%s&code=test-code", state),
		nil,
	))

	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusFound, res.Status)
	s.Require().NotNil(res.Redirect)
	s.Equal("https://app.example.com/dashboard", *res.Redirect)
	s.Require().Len(res.Cookies, 1)
	s.Equal(buildconf.SessionCookieName, res.Cookies[0].Name)
}

// Test_Callback_UnverifiedEmail_RedirectsWithError checks that a provider which
// returns an unverified email address is refused before any account is created
// or linked, closing the account-takeover-by-email-linking vector.
func (s *WithSKAuthOAuthSuite) Test_Callback_UnverifiedEmail_RedirectsWithError() {
	host := s.setupCallbackEnv(true)

	token := &oauth2.Token{}

	s.mockClient.On("Exchange", mock.Anything, mock.Anything).Return(token, nil).Once()
	s.mockClient.On("UserInfo", mock.Anything, token).Return(&skauth.UserInfo{
		AccountID:     "test-account-id",
		Email:         "jane@stormkit.io",
		EmailVerified: false,
		FirstName:     "Jane",
	}, nil).Once()

	state := s.stateToken(skauth.ProviderGoogle, "https://app.example.com/login")

	res, err := hosting.ServeAuth(s.oauthRequest(
		host,
		fmt.Sprintf("/_stormkit/auth/callback?state=%s&code=test-code", state),
		nil,
	))

	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusFound, res.Status)
	s.Require().NotNil(res.Redirect)
	s.Contains(*res.Redirect, "https://app.example.com/dashboard?login_error=")
	s.Contains(*res.Redirect, "verified")
	s.Empty(res.Cookies)
}

// Test_Callback_SetsCookieWithDomain verifies the OAuth-provider callback sets a
// first-party session cookie on this auth host (also the OAuth AS), scoped to the
// configured cookie domain, and redirects straight to the initiating origin — no
// dependency on that origin carrying its own auth config (the decoupled case).
func (s *WithSKAuthOAuthSuite) Test_Callback_SetsCookieWithDomain() {
	host := s.setupCallbackEnvWith(true, &buildconf.SKAuthConf{
		Secret:         "test-secret-padded-to-32-chars!!",
		Status:         true,
		CookieDomain:   ".example.com",
		AllowedOrigins: []string{"https://app.example.com"},
	})

	token := &oauth2.Token{}

	s.mockClient.On("Exchange", mock.Anything, mock.MatchedBy(func(req *shttp.RequestContext) bool {
		return req.FormValue("code") == "test-code"
	})).Return(token, nil).Once()

	s.mockClient.On("UserInfo", mock.Anything, token).Return(&skauth.UserInfo{
		AccountID:     "test-account-id",
		Email:         "jane@stormkit.io",
		EmailVerified: true,
		FirstName:     "Jane",
		LastName:      "Doe",
	}, nil).Once()

	state := s.stateToken(skauth.ProviderGoogle, "https://app.example.com/login")

	res, err := hosting.ServeAuth(s.oauthRequest(
		host,
		fmt.Sprintf("/_stormkit/auth/callback?state=%s&code=test-code", state),
		nil,
	))

	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusFound, res.Status)
	s.Require().NotNil(res.Redirect)
	s.Equal("https://app.example.com/dashboard", *res.Redirect)
	s.NotContains(*res.Redirect, "/_stormkit/auth?code=")

	s.Require().Len(res.Cookies, 1)
	s.Equal(buildconf.SessionCookieName, res.Cookies[0].Name)
	s.Equal(".example.com", res.Cookies[0].Domain)
	s.True(res.Cookies[0].HttpOnly)
	s.True(res.Cookies[0].Secure)
}

// Test_Callback_LinksToExistingMagicLinkUser guards against the duplicate-account
// failure that occurred when a user who first signed up through magic link later
// logged in through OAuth with the same email: the callback must link the OAuth
// provider to the existing user, not insert a second user (which violated the
// unique email constraint).
func (s *WithSKAuthOAuthSuite) Test_Callback_LinksToExistingMagicLinkUser() {
	host := s.setupCallbackEnv(true)

	conf := &buildconf.SchemaConf{
		SchemaName:        s.conn.Cfg.Schema,
		DBName:            s.conn.Cfg.DBName,
		Port:              s.conn.Cfg.Port,
		Host:              s.conn.Cfg.Host,
		MigrationUserName: s.conn.Cfg.User,
		MigrationPassword: s.conn.Cfg.Password,
		AppUserName:       s.conn.Cfg.User,
		AppPassword:       s.conn.Cfg.Password,
	}

	store, err := conf.Store(buildconf.SchemaAccessTypeAppUser)
	s.Require().NoError(err)

	// The user first registers through magic link with this email.
	_, err = store.UpsertMagicLinkUser(context.Background(), "jane@stormkit.io", "magic-token")
	s.Require().NoError(err)

	token := &oauth2.Token{}

	s.mockClient.On("Exchange", mock.Anything, mock.Anything).Return(token, nil).Once()
	s.mockClient.On("UserInfo", mock.Anything, token).Return(&skauth.UserInfo{
		AccountID:     "google-account-id",
		Email:         "jane@stormkit.io",
		EmailVerified: true,
		FirstName:     "Jane",
	}, nil).Once()

	state := s.stateToken(skauth.ProviderGoogle, "https://app.example.com/login")

	res, err := hosting.ServeAuth(s.oauthRequest(
		host,
		fmt.Sprintf("/_stormkit/auth/callback?state=%s&code=test-code", state),
		nil,
	))

	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusFound, res.Status)

	// The OAuth login linked to the existing user — no duplicate was created.
	users, err := store.ListAuthUsers(context.Background(), 0, 100)
	s.Require().NoError(err)

	matched := 0

	for _, u := range users {
		if u.Email == "jane@stormkit.io" {
			matched++
		}
	}

	s.Equal(1, matched)
}

func (s *WithSKAuthOAuthSuite) Test_Callback_InvalidState() {
	host := s.setupCallbackEnv(true)

	res, err := hosting.ServeAuth(s.oauthRequest(
		host,
		"/_stormkit/auth/callback?state=not-a-jwt&code=test-code",
		nil,
	))

	s.Require().NoError(err)
	s.Equal(http.StatusBadRequest, res.Status)
}

// Test_Callback_ForeignSecretState_Rejected proves a state token signed with a
// different secret (a different tenant, or a forgery attempt) does not verify
// and is rejected before any token exchange.
func (s *WithSKAuthOAuthSuite) Test_Callback_ForeignSecretState_Rejected() {
	host := s.setupCallbackEnv(true)

	foreign, err := user.JWT(jwt.MapClaims{
		"prv": skauth.ProviderGoogle,
		"ref": "https://app.example.com/login",
	}, "a-completely-different-tenant-secret!!")
	s.Require().NoError(err)

	res, err := hosting.ServeAuth(s.oauthRequest(
		host,
		fmt.Sprintf("/_stormkit/auth/callback?state=%s&code=test-code", foreign),
		nil,
	))

	s.Require().NoError(err)
	s.Equal(http.StatusBadRequest, res.Status)
}

// Test_Callback_UnlistedReferrer_FallsBackToOwnOrigin proves the callback never
// delivers the session to an origin outside the allow-list even if the (validly
// signed) state carries one — it stays first-party on the app's own origin.
func (s *WithSKAuthOAuthSuite) Test_Callback_UnlistedReferrer_FallsBackToOwnOrigin() {
	host := s.setupCallbackEnv(true)

	token := &oauth2.Token{}

	s.mockClient.On("Exchange", mock.Anything, mock.Anything).Return(token, nil).Once()
	s.mockClient.On("UserInfo", mock.Anything, token).Return(&skauth.UserInfo{
		AccountID:     "test-account-id",
		Email:         "jane@stormkit.io",
		EmailVerified: true,
		FirstName:     "Jane",
	}, nil).Once()

	state := s.stateToken(skauth.ProviderGoogle, "https://evil.example.com/login")

	res, err := hosting.ServeAuth(s.oauthRequest(
		host,
		fmt.Sprintf("/_stormkit/auth/callback?state=%s&code=test-code", state),
		nil,
	))

	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusFound, res.Status)
	s.Require().NotNil(res.Redirect)
	// Falls back to the app's own origin (the request host), never evil.example.com.
	s.Equal("https://api.example.com/dashboard", *res.Redirect)
	s.NotContains(*res.Redirect, "evil.example.com")
}

// Test_Callback_ProviderDisabled_RedirectsWithError checks that a disabled
// provider bounces the user back to the initiating origin with a friendly
// login_error message instead of rendering a Stormkit error page.
func (s *WithSKAuthOAuthSuite) Test_Callback_ProviderDisabled_RedirectsWithError() {
	host := s.setupCallbackEnv(false)

	state := s.stateToken(skauth.ProviderGoogle, "https://app.example.com/login")

	res, err := hosting.ServeAuth(s.oauthRequest(
		host,
		fmt.Sprintf("/_stormkit/auth/callback?state=%s&code=test-code", state),
		nil,
	))

	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusFound, res.Status)
	s.Require().NotNil(res.Redirect)
	s.Contains(*res.Redirect, "https://app.example.com/dashboard?login_error=")
	s.Contains(*res.Redirect, "not+available")
}

// Test_Callback_ProviderDenied_RedirectsWithError checks that when the provider
// round-trips an error param (e.g. the user denied consent), we bounce back with
// a friendly message and never attempt the token exchange.
func (s *WithSKAuthOAuthSuite) Test_Callback_ProviderDenied_RedirectsWithError() {
	host := s.setupCallbackEnv(true)

	state := s.stateToken(skauth.ProviderGoogle, "https://app.example.com/login")

	res, err := hosting.ServeAuth(s.oauthRequest(
		host,
		fmt.Sprintf("/_stormkit/auth/callback?state=%s&error=access_denied", state),
		nil,
	))

	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusFound, res.Status)
	s.Require().NotNil(res.Redirect)
	s.Contains(*res.Redirect, "https://app.example.com/dashboard?login_error=")
	s.Contains(*res.Redirect, "cancelled")
}

// Test_Callback_ExchangeFails_RedirectsWithError checks that a failed token
// exchange redirects the user back with a friendly message rather than a 500.
func (s *WithSKAuthOAuthSuite) Test_Callback_ExchangeFails_RedirectsWithError() {
	host := s.setupCallbackEnv(true)

	s.mockClient.On("Exchange", mock.Anything, mock.Anything).
		Return((*oauth2.Token)(nil), fmt.Errorf("invalid_grant")).Once()

	state := s.stateToken(skauth.ProviderGoogle, "https://app.example.com/login")

	res, err := hosting.ServeAuth(s.oauthRequest(
		host,
		fmt.Sprintf("/_stormkit/auth/callback?state=%s&code=bad-code", state),
		nil,
	))

	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusFound, res.Status)
	s.Require().NotNil(res.Redirect)
	s.Contains(*res.Redirect, "https://app.example.com/dashboard?login_error=")
	s.Contains(*res.Redirect, "complete+sign-in")
}

// Test_Callback_NativeScheme_TokenInRedirect covers the native/mobile OAuth
// path: the sign-in runs inside a system auth session whose cookie jar discards
// Set-Cookie, so an allow-listed custom-scheme target receives the session token
// in the redirect URL and no cookie at all.
func (s *WithSKAuthOAuthSuite) Test_Callback_NativeScheme_TokenInRedirect() {
	host := s.setupCallbackEnvWith(true, &buildconf.SKAuthConf{
		Secret:         "test-secret-padded-to-32-chars!!",
		Status:         true,
		AllowedOrigins: []string{"triplan://auth"},
	})

	token := &oauth2.Token{}

	s.mockClient.On("Exchange", mock.Anything, mock.Anything).Return(token, nil).Once()
	s.mockClient.On("UserInfo", mock.Anything, token).Return(&skauth.UserInfo{
		AccountID:     "native-account-id",
		Email:         "native@stormkit.io",
		EmailVerified: true,
		FirstName:     "Nat",
	}, nil).Once()

	state := s.stateToken(skauth.ProviderGoogle, "triplan://auth")

	res, err := hosting.ServeAuth(s.oauthRequest(
		host,
		fmt.Sprintf("/_stormkit/auth/callback?state=%s&code=test-code", state),
		nil,
	))

	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusFound, res.Status)
	s.Require().NotNil(res.Redirect)
	s.Empty(res.Cookies, "a native custom-scheme delivery must not set a cookie")

	target, err := url.Parse(*res.Redirect)
	s.Require().NoError(err)
	s.Equal("triplan", target.Scheme)
	s.Equal("auth", target.Host)
	s.Empty(target.Path, "the success URL must not be appended to a deep link")

	claims := user.ParseJWT(&user.ParseJWTArgs{
		Bearer: target.Query().Get("token"),
		Secret: "test-secret-padded-to-32-chars!!",
	})

	s.Require().NotNil(claims, "the redirected token must be a valid session JWT")
	s.NotEmpty(claims["uid"])
}

// Test_Callback_NativeScheme_NotAllowListed_StaysFirstParty guards the token
// leak: a custom scheme that is not on the allow-list must not receive a token,
// and the flow falls back to a first-party cookie on this host.
func (s *WithSKAuthOAuthSuite) Test_Callback_NativeScheme_NotAllowListed_StaysFirstParty() {
	host := s.setupCallbackEnv(true)

	token := &oauth2.Token{}

	s.mockClient.On("Exchange", mock.Anything, mock.Anything).Return(token, nil).Once()
	s.mockClient.On("UserInfo", mock.Anything, token).Return(&skauth.UserInfo{
		AccountID:     "evil-account-id",
		Email:         "evil@stormkit.io",
		EmailVerified: true,
		FirstName:     "Eve",
	}, nil).Once()

	state := s.stateToken(skauth.ProviderGoogle, "evil://auth")

	res, err := hosting.ServeAuth(s.oauthRequest(
		host,
		fmt.Sprintf("/_stormkit/auth/callback?state=%s&code=test-code", state),
		nil,
	))

	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusFound, res.Status)
	s.Require().NotNil(res.Redirect)
	s.Equal("https://api.example.com/dashboard", *res.Redirect)
	s.NotContains(*res.Redirect, "token=")
	s.Require().Len(res.Cookies, 1, "the fallback must stay on the first-party cookie")
	s.Equal(buildconf.SessionCookieName, res.Cookies[0].Name)
}
