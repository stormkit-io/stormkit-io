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
			Secret:         "test-secret",
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
				Secret:     "test-secret",
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

	res, err := hosting.WithSKAuth(s.magicRequest(s.hostFor(env.ID), "/_stormkit/auth/magic?email=not-an-email"))

	s.NoError(err)
	s.Equal(http.StatusBadRequest, res.Status)
}

func (s *WithSKAuthMagicLinkSuite) Test_Request_ProviderNotEnabled() {
	env := s.MockEnv(s.app, nil)

	res, err := hosting.WithSKAuth(s.magicRequest(s.hostFor(env.ID), "/_stormkit/auth/magic?email=user@example.com"))

	s.NoError(err)
	s.Equal(http.StatusNotFound, res.Status)
}

func (s *WithSKAuthMagicLinkSuite) Test_Request_Success() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	res, err := hosting.WithSKAuth(s.magicRequest(s.hostFor(env.ID), "/_stormkit/auth/magic?email=user@example.com"))

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

	_, err = hosting.WithSKAuth(s.magicRequest(s.hostFor(env.ID), "/_stormkit/auth/magic?email=newuser@example.com"))
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

func (s *WithSKAuthMagicLinkSuite) Test_Verify_InvalidToken() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	res, err := hosting.WithSKAuth(s.magicRequest(s.hostFor(env.ID), "/_stormkit/auth/magic?token=bad-token"))

	s.NoError(err)
	s.Equal(http.StatusBadRequest, res.Status)
}

func (s *WithSKAuthMagicLinkSuite) Test_Verify_Success() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	_, err = hosting.WithSKAuth(s.magicRequest(s.hostFor(env.ID), "/_stormkit/auth/magic?email=verify@example.com"))
	s.Require().NoError(err)

	token := s.captureToken(env.ID)
	s.Require().NotEmpty(token)

	res, err := hosting.WithSKAuth(s.magicRequest(s.hostFor(env.ID), fmt.Sprintf("/_stormkit/auth/magic?token=%s", token)))

	s.NoError(err)
	s.Equal(http.StatusOK, res.Status)
	s.Contains(string(res.Data.([]byte)), "localStorage.setItem('skauth'")
}

// Test_Verify_UIDClaimIsUUID confirms the session JWT carries the user's
// UUID (not the internal numeric id) in the uid claim, which is what gets
// forwarded to backend apps as the X-User-Id header.
func (s *WithSKAuthMagicLinkSuite) Test_Verify_UIDClaimIsUUID() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	_, err = hosting.WithSKAuth(s.magicRequest(s.hostFor(env.ID), "/_stormkit/auth/magic?email=uuidcheck@example.com"))
	s.Require().NoError(err)

	token := s.captureToken(env.ID)
	s.Require().NotEmpty(token)

	res, err := hosting.WithSKAuth(s.magicRequest(s.hostFor(env.ID), fmt.Sprintf("/_stormkit/auth/magic?token=%s", token)))
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, res.Status)

	body := string(res.Data.([]byte))
	const marker = `JSON.stringify("`
	idx := strings.Index(body, marker)
	s.Require().GreaterOrEqual(idx, 0, "session token missing from response body")

	start := idx + len(marker)
	end := start

	for end < len(body) && body[end] != '"' {
		end++
	}

	sessionToken := body[start:end]
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

	_, err = hosting.WithSKAuth(s.magicRequest(s.hostFor(env.ID), "/_stormkit/auth/magic?email=once@example.com"))
	s.Require().NoError(err)

	token := s.captureToken(env.ID)
	s.Require().NotEmpty(token)

	path := fmt.Sprintf("/_stormkit/auth/magic?token=%s", token)

	first, err := hosting.WithSKAuth(s.magicRequest(s.hostFor(env.ID), path))
	s.NoError(err)
	s.Equal(http.StatusOK, first.Status)

	second, err := hosting.WithSKAuth(s.magicRequest(s.hostFor(env.ID), path))
	s.NoError(err)
	s.Equal(http.StatusBadRequest, second.Status)
}

func (s *WithSKAuthMagicLinkSuite) Test_Request_NoOriginCheck_WhenAllowListEmpty() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	rq := s.magicRequestWith(s.hostFor(env.ID), "/_stormkit/auth/magic?email=user@example.com", http.MethodPost, "https://other.example.com")
	res, err := hosting.WithSKAuth(rq)

	s.NoError(err)
	s.Equal(http.StatusCreated, res.Status)
}

func (s *WithSKAuthMagicLinkSuite) Test_Request_AllowedOrigin_Accepted() {
	env, err := s.setupEnvWithOrigins([]string{"https://app.example.com"})
	s.Require().NoError(err)

	rq := s.magicRequestWith(s.hostFor(env.ID), "/_stormkit/auth/magic?email=user@example.com", http.MethodPost, "https://app.example.com")
	res, err := hosting.WithSKAuth(rq)

	s.NoError(err)
	s.Equal(http.StatusCreated, res.Status)
}

func (s *WithSKAuthMagicLinkSuite) Test_Request_DisallowedOrigin_Returns403() {
	env, err := s.setupEnvWithOrigins([]string{"https://app.example.com"})
	s.Require().NoError(err)

	rq := s.magicRequestWith(s.hostFor(env.ID), "/_stormkit/auth/magic?email=user@example.com", http.MethodPost, "https://evil.example.com")
	res, err := hosting.WithSKAuth(rq)

	s.NoError(err)
	s.Equal(http.StatusForbidden, res.Status)
}

func (s *WithSKAuthMagicLinkSuite) Test_Verify_CrossOrigin_RedirectsToInitiator() {
	env, err := s.setupEnvWithOrigins([]string{"https://app.example.com"})
	s.Require().NoError(err)

	post := s.magicRequestWith(s.hostFor(env.ID), "/_stormkit/auth/magic?email=cross@example.com", http.MethodPost, "https://app.example.com")
	_, err = hosting.WithSKAuth(post)
	s.Require().NoError(err)

	token := s.captureToken(env.ID)
	s.Require().NotEmpty(token)

	res, err := hosting.WithSKAuth(s.magicRequest(s.hostFor(env.ID), fmt.Sprintf("/_stormkit/auth/magic?token=%s", token)))

	s.NoError(err)
	s.Equal(http.StatusOK, res.Status)

	body := string(res.Data.([]byte))
	s.Contains(body, "https://app.example.com/dashboard#skauth=")
	// Cross-origin path must not touch localStorage on the wrong host.
	s.NotContains(body, "localStorage")
}

func (s *WithSKAuthMagicLinkSuite) Test_Verify_StaleRedirect_FallsBackToLocal() {
	env, err := s.setupEnvWithOrigins([]string{"https://app.example.com"})
	s.Require().NoError(err)

	post := s.magicRequestWith(s.hostFor(env.ID), "/_stormkit/auth/magic?email=stale@example.com", http.MethodPost, "https://app.example.com")
	_, err = hosting.WithSKAuth(post)
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

	res, err := hosting.WithSKAuth(s.magicRequest(s.hostFor(env.ID), fmt.Sprintf("/_stormkit/auth/magic?token=%s", token)))

	s.NoError(err)
	s.Equal(http.StatusOK, res.Status)

	body := string(res.Data.([]byte))
	s.Contains(body, "localStorage.setItem('skauth'")
	s.NotContains(body, "https://app.example.com/dashboard#skauth=")
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
