package hosting_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type WithSKAuthMagicLinkSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	app  *factory.MockApp
}

func (s *WithSKAuthMagicLinkSuite) BeforeTest(suiteName, _ string) {
	truncateMagicLinkAuthTables()
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.app = s.MockApp(s.MockUser(nil), nil)
}

func (s *WithSKAuthMagicLinkSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *WithSKAuthMagicLinkSuite) setupEnv() (*factory.MockEnv, error) {
	return s.setupEnvWithOrigins(nil)
}

func (s *WithSKAuthMagicLinkSuite) setupEnvWithOrigins(allowed []string) (*factory.MockEnv, error) {
	env := s.MockEnv(s.app, map[string]any{
		"AuthConf": &buildconf.SKAuthConf{
			Secret:         "test-secret-padded-to-32-chars!!",
			Status:         true,
			AllowedOrigins: allowed,
		},
		"SchemaConf": &buildconf.SchemaConf{
			SchemaName:        s.conn.Cfg.Schema,
			DBName:            s.conn.Cfg.DBName,
			Port:              s.conn.Cfg.Port,
			Host:              s.conn.Cfg.Host,
			MigrationUserName: s.conn.Cfg.User,
			MigrationPassword: s.conn.Cfg.Password,
			AppUserName:       s.conn.Cfg.User,
			AppPassword:       s.conn.Cfg.Password,
			DriverName:        s.conn.Cfg.DriverName,
		},
	})

	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeMigrations)

	if err != nil {
		return nil, err
	}

	if err := store.CreateAuthTable(context.Background()); err != nil {
		return nil, err
	}

	err = skauth.NewStore().SaveProvider(context.Background(), skauth.SaveProviderArgs{
		EnvID: env.ID,
		AppID: s.app.ID,
		Provider: &skauth.Provider{
			Name:   skauth.ProviderMagicLink,
			Status: true,
		},
	})

	return env, err
}

func (s *WithSKAuthMagicLinkSuite) hostFor(envID types.ID) *hosting.Host {
	return &hosting.Host{
		Name: "my.example.com",
		Config: &appconf.Config{
			EnvID: envID,
			SKAuth: &buildconf.SKAuthConf{
				Secret:     "test-secret-padded-to-32-chars!!",
				SuccessURL: "/dashboard",
			},
		},
	}
}

func (s *WithSKAuthMagicLinkSuite) magicRequest(host *hosting.Host, path string) *hosting.RequestContext {
	return s.magicRequestWith(host, path, http.MethodGet, "")
}

func (s *WithSKAuthMagicLinkSuite) magicRequestWith(host *hosting.Host, path, method, origin string) *hosting.RequestContext {
	pieces := strings.SplitN(path, "?", 2)
	rawPath := pieces[0]
	query := ""

	if len(pieces) > 1 {
		query = pieces[1]
	}

	header := make(http.Header)

	if origin != "" {
		header.Set("Origin", origin)
	}

	rq := &hosting.RequestContext{
		Host: host,
		RequestContext: shttp.NewRequestContext(&http.Request{
			Method: method,
			Header: header,
			URL: &url.URL{
				Scheme:   "https",
				Host:     host.Name,
				Path:     rawPath,
				RawQuery: query,
				RawPath:  rawPath,
			},
		}),
	}

	rq.OriginalPath = rawPath

	return rq
}

func (s *WithSKAuthMagicLinkSuite) captureToken(envID types.ID) string {
	emails, err := buildconf.MailerStore().Emails(context.Background(), envID)
	s.Require().NoError(err)
	s.Require().NotEmpty(emails, "expected at least one stored email")

	body := emails[len(emails)-1].Body
	needle := "token="
	idx := strings.Index(body, needle)
	s.Require().GreaterOrEqual(idx, 0, "token not found in email body")

	start := idx + len(needle)
	end := start

	for end < len(body) && body[end] != '"' && body[end] != '<' && body[end] != '>' && body[end] != ' ' {
		end++
	}

	return body[start:end]
}

func (s *WithSKAuthMagicLinkSuite) Test_Request_InvalidEmail() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	res, err := hosting.ServeAuth(s.magicRequest(s.hostFor(env.ID), "/_stormkit/auth/magic?email=not-an-email"))

	s.NoError(err)
	s.Equal(http.StatusBadRequest, res.Status)
}

func (s *WithSKAuthMagicLinkSuite) Test_Request_ProviderNotEnabled() {
	env := s.MockEnv(s.app, nil)

	res, err := hosting.ServeAuth(s.magicRequest(s.hostFor(env.ID), "/_stormkit/auth/magic?email=user@example.com"))

	s.NoError(err)
	s.Equal(http.StatusNotFound, res.Status)
}

func (s *WithSKAuthMagicLinkSuite) Test_Request_Success() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	res, err := hosting.ServeAuth(s.magicRequest(s.hostFor(env.ID), "/_stormkit/auth/magic?email=user@example.com"))

	s.NoError(err)
	s.Equal(http.StatusCreated, res.Status)

	emails, err := buildconf.MailerStore().Emails(context.Background(), env.ID)
	s.Require().NoError(err)
	s.Require().Len(emails, 1)
	s.Equal("user@example.com", emails[0].To)
	s.Contains(emails[0].Body, "_stormkit/auth/magic?token=")
}

func (s *WithSKAuthMagicLinkSuite) Test_Request_CreatesNewUser() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	_, err = hosting.ServeAuth(s.magicRequest(s.hostFor(env.ID), "/_stormkit/auth/magic?email=newuser@example.com"))
	s.Require().NoError(err)

	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeAppUser)
	s.Require().NoError(err)

	authUser, err := store.AuthUserByEmail(context.Background(), buildconf.AuthUserByEmailParams{
		Email:    "newuser@example.com",
		Provider: skauth.ProviderMagicLink,
	})
	s.Require().NoError(err)
	s.Require().NotNil(authUser)
	s.Equal("newuser@example.com", authUser.Email)
}

// Test_Verify_InvalidToken checks that an unparseable token bounces the user back
// to the app's redirect page with a login_error message. The origin is unknown
// (the token can't be parsed), so we fall back to this host's own origin.
func (s *WithSKAuthMagicLinkSuite) Test_Verify_InvalidToken() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	res, err := hosting.ServeAuth(s.magicRequest(s.hostFor(env.ID), "/_stormkit/auth/magic?token=bad-token"))

	s.NoError(err)
	s.Require().NotNil(res)
	s.Equal(http.StatusFound, res.Status)
	s.Require().NotNil(res.Redirect)
	s.Contains(*res.Redirect, "https://my.example.com/dashboard?login_error=")
}

func (s *WithSKAuthMagicLinkSuite) Test_Verify_Success() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	_, err = hosting.ServeAuth(s.magicRequest(s.hostFor(env.ID), "/_stormkit/auth/magic?email=verify@example.com"))
	s.Require().NoError(err)

	token := s.captureToken(env.ID)
	s.Require().NotEmpty(token)

	res, err := hosting.ServeAuth(s.magicRequest(s.hostFor(env.ID), fmt.Sprintf("/_stormkit/auth/magic?token=%s", token)))

	s.NoError(err)
	s.Equal(http.StatusFound, res.Status)
	s.Require().Len(res.Cookies, 1)
	s.Equal(buildconf.SessionCookieName, res.Cookies[0].Name)
}

// Test_Verify_UIDClaimIsUUID confirms the session JWT carries the user's
// UUID (not the internal numeric id) in the uid claim, which is what gets
// forwarded to backend apps as the X-User-Id header.
func (s *WithSKAuthMagicLinkSuite) Test_Verify_UIDClaimIsUUID() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	_, err = hosting.ServeAuth(s.magicRequest(s.hostFor(env.ID), "/_stormkit/auth/magic?email=uuidcheck@example.com"))
	s.Require().NoError(err)

	token := s.captureToken(env.ID)
	s.Require().NotEmpty(token)

	res, err := hosting.ServeAuth(s.magicRequest(s.hostFor(env.ID), fmt.Sprintf("/_stormkit/auth/magic?token=%s", token)))
	s.Require().NoError(err)
	s.Require().Equal(http.StatusFound, res.Status)
	s.Require().Len(res.Cookies, 1)

	sessionToken := res.Cookies[0].Value
	claims := user.ParseJWT(&user.ParseJWTArgs{
		Bearer:  sessionToken,
		Secret:  env.AuthConf.Secret,
		MaxMins: 0,
	})

	s.Require().NotNil(claims, "session token should be parseable")

	uid, ok := claims["uid"].(string)
	s.Require().True(ok, "uid claim should be a string")
	_, err = uuid.Parse(uid)
	s.NoError(err, "uid claim should be a valid UUID, got %q", uid)
}

func (s *WithSKAuthMagicLinkSuite) Test_Verify_TokenConsumedOnce() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	_, err = hosting.ServeAuth(s.magicRequest(s.hostFor(env.ID), "/_stormkit/auth/magic?email=once@example.com"))
	s.Require().NoError(err)

	token := s.captureToken(env.ID)
	s.Require().NotEmpty(token)

	path := fmt.Sprintf("/_stormkit/auth/magic?token=%s", token)

	first, err := hosting.ServeAuth(s.magicRequest(s.hostFor(env.ID), path))
	s.NoError(err)
	s.Equal(http.StatusFound, first.Status)

	// Reusing a consumed token bounces back to the app with a login_error.
	second, err := hosting.ServeAuth(s.magicRequest(s.hostFor(env.ID), path))
	s.NoError(err)
	s.Equal(http.StatusFound, second.Status)
	s.Require().NotNil(second.Redirect)
	s.Contains(*second.Redirect, "/dashboard?login_error=")
}

// Test_Verify_Error_CrossOrigin_RedirectsToInitiator checks that a failed verify
// for a cross-origin flow lands the login_error on the initiating origin (carried
// in the token) rather than this host.
func (s *WithSKAuthMagicLinkSuite) Test_Verify_Error_CrossOrigin_RedirectsToInitiator() {
	env, err := s.setupEnvWithOrigins([]string{"https://app.example.com"})
	s.Require().NoError(err)

	post := s.magicRequestWith(s.hostFor(env.ID), "/_stormkit/auth/magic?email=err@example.com", http.MethodPost, "https://app.example.com")
	_, err = hosting.ServeAuth(post)
	s.Require().NoError(err)

	token := s.captureToken(env.ID)
	s.Require().NotEmpty(token)

	path := fmt.Sprintf("/_stormkit/auth/magic?token=%s", token)

	first, err := hosting.ServeAuth(s.magicRequest(s.hostFor(env.ID), path))
	s.Require().NoError(err)
	s.Require().Equal(http.StatusFound, first.Status)

	// Replaying the consumed token: the error redirect must target the initiating
	// origin, not the verify host.
	second, err := hosting.ServeAuth(s.magicRequest(s.hostFor(env.ID), path))
	s.NoError(err)
	s.Equal(http.StatusFound, second.Status)
	s.Require().NotNil(second.Redirect)
	s.Contains(*second.Redirect, "https://app.example.com/dashboard?login_error=")
}

func (s *WithSKAuthMagicLinkSuite) Test_Request_NoOriginCheck_WhenAllowListEmpty() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	rq := s.magicRequestWith(s.hostFor(env.ID), "/_stormkit/auth/magic?email=user@example.com", http.MethodPost, "https://other.example.com")
	res, err := hosting.ServeAuth(rq)

	s.NoError(err)
	s.Equal(http.StatusCreated, res.Status)
}

func (s *WithSKAuthMagicLinkSuite) Test_Request_AllowedOrigin_Accepted() {
	env, err := s.setupEnvWithOrigins([]string{"https://app.example.com"})
	s.Require().NoError(err)

	rq := s.magicRequestWith(s.hostFor(env.ID), "/_stormkit/auth/magic?email=user@example.com", http.MethodPost, "https://app.example.com")
	res, err := hosting.ServeAuth(rq)

	s.NoError(err)
	s.Equal(http.StatusCreated, res.Status)
}

func (s *WithSKAuthMagicLinkSuite) Test_Request_DisallowedOrigin_Returns403() {
	env, err := s.setupEnvWithOrigins([]string{"https://app.example.com"})
	s.Require().NoError(err)

	rq := s.magicRequestWith(s.hostFor(env.ID), "/_stormkit/auth/magic?email=user@example.com", http.MethodPost, "https://evil.example.com")
	res, err := hosting.ServeAuth(rq)

	s.NoError(err)
	s.Equal(http.StatusForbidden, res.Status)
}

func (s *WithSKAuthMagicLinkSuite) Test_Verify_CrossOrigin_RedirectsToInitiator() {
	env, err := s.setupEnvWithOrigins([]string{"https://app.example.com"})
	s.Require().NoError(err)

	post := s.magicRequestWith(s.hostFor(env.ID), "/_stormkit/auth/magic?email=cross@example.com", http.MethodPost, "https://app.example.com")
	_, err = hosting.ServeAuth(post)
	s.Require().NoError(err)

	token := s.captureToken(env.ID)
	s.Require().NotEmpty(token)

	res, err := hosting.ServeAuth(s.magicRequest(s.hostFor(env.ID), fmt.Sprintf("/_stormkit/auth/magic?token=%s", token)))

	s.NoError(err)
	// Cross-origin: the session cookie is set first-party on this auth host and
	// the browser is 302'd straight to the initiating origin's success URL.
	s.Equal(http.StatusFound, res.Status)
	s.Require().NotNil(res.Redirect)
	s.Equal("https://app.example.com/dashboard", *res.Redirect)
	s.Require().Len(res.Cookies, 1)
	s.Equal(buildconf.SessionCookieName, res.Cookies[0].Name)
}

func (s *WithSKAuthMagicLinkSuite) Test_Verify_StaleRedirect_FallsBackToLocal() {
	env, err := s.setupEnvWithOrigins([]string{"https://app.example.com"})
	s.Require().NoError(err)

	post := s.magicRequestWith(s.hostFor(env.ID), "/_stormkit/auth/magic?email=stale@example.com", http.MethodPost, "https://app.example.com")
	_, err = hosting.ServeAuth(post)
	s.Require().NoError(err)

	token := s.captureToken(env.ID)
	s.Require().NotEmpty(token)

	// Allow-list changes between POST and verify — drop the previously
	// whitelisted origin. The stale rdr claim must not be trusted.
	envStore := buildconf.NewStore()
	stored, err := envStore.EnvironmentByID(context.Background(), env.ID)
	s.Require().NoError(err)
	stored.AuthConf.AllowedOrigins = nil
	s.Require().NoError(envStore.SaveAuthConf(context.Background(), env.ID, stored.AuthConf))

	res, err := hosting.ServeAuth(s.magicRequest(s.hostFor(env.ID), fmt.Sprintf("/_stormkit/auth/magic?token=%s", token)))

	s.NoError(err)
	// Stale redirect is dropped, so the session is delivered same-origin: cookie
	// set here and 302 to the local success URL, never the untrusted origin.
	s.Equal(http.StatusFound, res.Status)
	s.Require().NotNil(res.Redirect)
	s.Equal("/dashboard?verified=true", *res.Redirect)
	s.Require().Len(res.Cookies, 1)
	s.Equal(buildconf.SessionCookieName, res.Cookies[0].Name)
}

// Test_Verify_NativeScheme_TokenInRedirect covers the native/mobile path: a
// custom-scheme redirect target has no usable cookie jar (the OS auth session
// discards Set-Cookie), so the session token rides the redirect URL instead and
// no cookie is set.
func (s *WithSKAuthMagicLinkSuite) Test_Verify_NativeScheme_TokenInRedirect() {
	env, err := s.setupEnvWithOrigins([]string{"triplan://auth"})
	s.Require().NoError(err)

	post := s.magicRequestWith(
		s.hostFor(env.ID),
		"/_stormkit/auth/magic?email=native@example.com&redirect=triplan://auth",
		http.MethodPost,
		"",
	)
	_, err = hosting.ServeAuth(post)
	s.Require().NoError(err)

	token := s.captureToken(env.ID)
	s.Require().NotEmpty(token)

	res, err := hosting.ServeAuth(s.magicRequest(s.hostFor(env.ID), fmt.Sprintf("/_stormkit/auth/magic?token=%s", token)))

	s.NoError(err)
	s.Equal(http.StatusFound, res.Status)
	s.Require().NotNil(res.Redirect)
	s.Empty(res.Cookies, "a native custom-scheme delivery must not set a cookie")

	target, err := url.Parse(*res.Redirect)
	s.Require().NoError(err)
	s.Equal("triplan", target.Scheme)
	s.Equal("auth", target.Host)
	s.Empty(target.Path, "the success URL must not be appended to a deep link")

	// The token in the URL is the same session JWT the cookie would have carried.
	claims := user.ParseJWT(&user.ParseJWTArgs{
		Bearer: target.Query().Get("token"),
		Secret: env.AuthConf.Secret,
	})

	s.Require().NotNil(claims, "the redirected token must be a valid session JWT")

	uid, ok := claims["uid"].(string)
	s.Require().True(ok)
	_, err = uuid.Parse(uid)
	s.NoError(err, "uid claim should be a valid UUID, got %q", uid)
}

// Test_Verify_NativeScheme_ErrorRedirect verifies a failed sign-in bounces back
// to the deep link itself — there is no success page to land on in a native app.
func (s *WithSKAuthMagicLinkSuite) Test_Verify_NativeScheme_ErrorRedirect() {
	env, err := s.setupEnvWithOrigins([]string{"triplan://auth"})
	s.Require().NoError(err)

	post := s.magicRequestWith(
		s.hostFor(env.ID),
		"/_stormkit/auth/magic?email=nativeerr@example.com&redirect=triplan://auth",
		http.MethodPost,
		"",
	)
	_, err = hosting.ServeAuth(post)
	s.Require().NoError(err)

	token := s.captureToken(env.ID)
	path := fmt.Sprintf("/_stormkit/auth/magic?token=%s", token)

	first, err := hosting.ServeAuth(s.magicRequest(s.hostFor(env.ID), path))
	s.Require().NoError(err)
	s.Require().Equal(http.StatusFound, first.Status)

	// Replaying the consumed token must land the error on the deep link.
	second, err := hosting.ServeAuth(s.magicRequest(s.hostFor(env.ID), path))

	s.NoError(err)
	s.Equal(http.StatusFound, second.Status)
	s.Require().NotNil(second.Redirect)
	s.Contains(*second.Redirect, "triplan://auth?login_error=")
	s.NotContains(*second.Redirect, "/dashboard")
}

// Test_Request_NativeScheme_NotAllowListed_Returns403 guards the token-leak
// vector: an unregistered custom scheme must never become a delivery target.
func (s *WithSKAuthMagicLinkSuite) Test_Request_NativeScheme_NotAllowListed_Returns403() {
	env, err := s.setupEnvWithOrigins([]string{"https://app.example.com"})
	s.Require().NoError(err)

	rq := s.magicRequestWith(
		s.hostFor(env.ID),
		"/_stormkit/auth/magic?email=evil@example.com&redirect=evil://auth",
		http.MethodPost,
		"",
	)
	res, err := hosting.ServeAuth(rq)

	s.NoError(err)
	s.Equal(http.StatusForbidden, res.Status)
}

func truncateMagicLinkAuthTables() {
	db, err := sql.Open("postgres", database.ConnectionString(database.Config))

	if err != nil {
		return
	}

	defer db.Close()

	if _, err := db.Exec(`
		DO $$
		BEGIN
			IF EXISTS (SELECT FROM pg_tables WHERE schemaname = current_schema() AND tablename = 'stormkit_auth_users') THEN
				TRUNCATE stormkit_auth_users, stormkit_auth_providers RESTART IDENTITY CASCADE;
			END IF;
		END $$;
	`); err != nil {
		panic("truncateMagicLinkAuthTables: " + err.Error())
	}
}

func TestWithSKAuthMagicLink(t *testing.T) {
	suite.Run(t, new(WithSKAuthMagicLinkSuite))
}
