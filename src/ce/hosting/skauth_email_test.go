package hosting_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type WithSKAuthEmailSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	app  *factory.MockApp
}

func (s *WithSKAuthEmailSuite) BeforeTest(suiteName, _ string) {
	truncateMagicLinkAuthTables()
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.app = s.MockApp(s.MockUser(nil), nil)
	buildconf.SendMailFunc = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		return nil
	}
}

func (s *WithSKAuthEmailSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	buildconf.SendMailFunc = buildconf.SendMailWithDeadline
}

func (s *WithSKAuthEmailSuite) hostFor(envID types.ID) *hosting.Host {
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

func (s *WithSKAuthEmailSuite) hostForCookie(envID types.ID) *hosting.Host {
	host := s.hostFor(envID)
	host.Config.SKAuth.CookieDomain = ".example.com"

	return host
}

func (s *WithSKAuthEmailSuite) postRequest(host *hosting.Host, path string, body any) *hosting.RequestContext {
	var buf bytes.Buffer

	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}

	pieces := strings.SplitN(path, "?", 2)
	rawPath := pieces[0]
	query := ""

	if len(pieces) > 1 {
		query = pieces[1]
	}

	rq := &hosting.RequestContext{
		Host: host,
		RequestContext: shttp.NewRequestContext(&http.Request{
			Method: http.MethodPost,
			Header: make(http.Header),
			URL: &url.URL{
				Scheme:   "https",
				Host:     host.Name,
				Path:     rawPath,
				RawQuery: query,
				RawPath:  rawPath,
			},
			Body: io.NopCloser(&buf),
		}),
	}

	rq.OriginalPath = rawPath

	return rq
}

func (s *WithSKAuthEmailSuite) getRequest(host *hosting.Host, path string) *hosting.RequestContext {
	pieces := strings.SplitN(path, "?", 2)
	rawPath := pieces[0]
	query := ""

	if len(pieces) > 1 {
		query = pieces[1]
	}

	rq := &hosting.RequestContext{
		Host: host,
		RequestContext: shttp.NewRequestContext(&http.Request{
			Method: http.MethodGet,
			Header: make(http.Header),
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

func (s *WithSKAuthEmailSuite) setupEnv(withMailer bool) (*factory.MockEnv, error) {
	fields := map[string]any{
		"AuthConf": &buildconf.SKAuthConf{
			Secret:     "test-secret-padded-to-32-chars!!",
			Status:     true,
			SuccessURL: "/dashboard",
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
	}

	if withMailer {
		fields["MailerConf"] = &buildconf.MailerConf{
			Host:     "smtp.example.com",
			Port:     "587",
			Username: "test@example.com",
			Password: "secret",
		}
	}

	env := s.MockEnv(s.app, fields)

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
			Name:   skauth.ProviderEmail,
			Status: true,
		},
	})

	return env, err
}

// register and login default to bearer delivery (X-Session-Delivery: bearer), so
// the response carries the token in the body and skips the cookie-CSRF Origin
// gate — the simplest shape for tests that just need a session or a user. The
// browser cookie path is exercised by registerOrigin / loginOrigin.
func (s *WithSKAuthEmailSuite) register(host *hosting.Host, email, password string) *shttp.Response {
	req := s.postRequest(host, "/_stormkit/auth/register", map[string]any{
		"email":    email,
		"password": password,
	})
	req.Header.Set("X-Session-Delivery", "bearer")

	res, err := hosting.ServeAuth(req)
	s.Require().NoError(err)
	return res
}

func (s *WithSKAuthEmailSuite) login(host *hosting.Host, email, password string) *shttp.Response {
	req := s.postRequest(host, "/_stormkit/auth/login", map[string]any{
		"email":    email,
		"password": password,
	})
	req.Header.Set("X-Session-Delivery", "bearer")

	res, err := hosting.ServeAuth(req)
	s.Require().NoError(err)
	return res
}

// sameHostOrigin is the Origin a first-party browser POST carries against the
// auth host — the value the cookie-mode CSRF guard accepts.
func (s *WithSKAuthEmailSuite) sameHostOrigin(host *hosting.Host) string {
	return "https://" + host.Name
}

// loginBearer logs in as a native/mobile client: it opts into bearer delivery
// via the X-Session-Delivery header and sends no Origin (native clients aren't
// subject to CORS), so it must receive the token in the body and no cookie.
func (s *WithSKAuthEmailSuite) loginBearer(host *hosting.Host, email, password string) *shttp.Response {
	req := s.postRequest(host, "/_stormkit/auth/login", map[string]any{
		"email":    email,
		"password": password,
	})
	req.Header.Set("X-Session-Delivery", "bearer")

	res, err := hosting.ServeAuth(req)
	s.Require().NoError(err)
	return res
}

func (s *WithSKAuthEmailSuite) loginOrigin(host *hosting.Host, origin, email, password string) *shttp.Response {
	req := s.postRequest(host, "/_stormkit/auth/login", map[string]any{
		"email":    email,
		"password": password,
	})
	req.Header.Set("Origin", origin)

	res, err := hosting.ServeAuth(req)
	s.Require().NoError(err)
	return res
}

func (s *WithSKAuthEmailSuite) registerOrigin(host *hosting.Host, origin, email, password string) *shttp.Response {
	req := s.postRequest(host, "/_stormkit/auth/register", map[string]any{
		"email":    email,
		"password": password,
	})
	req.Header.Set("Origin", origin)

	res, err := hosting.ServeAuth(req)
	s.Require().NoError(err)
	return res
}

func (s *WithSKAuthEmailSuite) jsonData(res *shttp.Response) map[string]any {
	b, err := json.Marshal(res.Data)
	s.Require().NoError(err)

	var m map[string]any
	s.Require().NoError(json.Unmarshal(b, &m))

	return m
}

func (s *WithSKAuthEmailSuite) errors(res *shttp.Response) []any {
	d := s.jsonData(res)
	errs, _ := d["errors"].([]any)
	return errs
}

// Register tests

// Test_Register_BodyEnvIdIgnored verifies that a client-supplied envId in the request
// body is silently ignored; the host's own EnvID is always used to prevent cross-tenant access.
func (s *WithSKAuthEmailSuite) Test_Register_BodyEnvIdIgnored() {
	env, err := s.setupEnv(false)
	s.Require().NoError(err)

	// Post with a different envId in the body — should still succeed using the host's env.
	req := s.postRequest(s.hostFor(env.ID), "/_stormkit/auth/register", map[string]any{
		"envId":    "9999999",
		"email":    "crossenv@example.com",
		"password": "supersecret123",
	})
	req.Header.Set("X-Session-Delivery", "bearer")

	res, err := hosting.ServeAuth(req)
	s.Require().NoError(err)
	s.Equal(http.StatusCreated, res.Status)
}

func (s *WithSKAuthEmailSuite) Test_Register_Success() {
	env, err := s.setupEnv(false)
	s.Require().NoError(err)

	res := s.register(s.hostFor(env.ID), "register@example.com", "supersecret123")

	s.Equal(http.StatusCreated, res.Status)

	data := s.jsonData(res)
	s.NotEmpty(data["token"])
	s.Equal("register@example.com", data["email"])
	s.NotEmpty(data["userId"])
}

func (s *WithSKAuthEmailSuite) Test_Register_DuplicateEmail() {
	env, err := s.setupEnv(false)
	s.Require().NoError(err)

	host := s.hostFor(env.ID)

	s.Require().Equal(http.StatusCreated, s.register(host, "dup@example.com", "supersecret123").Status)

	res := s.register(host, "dup@example.com", "supersecret123")

	s.Equal(http.StatusBadRequest, res.Status)
	s.Equal("an account with this email already exists", s.errors(res)[0])
}

func (s *WithSKAuthEmailSuite) Test_Register_InvalidEmail() {
	env, err := s.setupEnv(false)
	s.Require().NoError(err)

	res := s.register(s.hostFor(env.ID), "not-an-email", "supersecret123")

	s.Equal(http.StatusBadRequest, res.Status)
	s.Equal("email is invalid", s.errors(res)[0])
}

func (s *WithSKAuthEmailSuite) Test_Register_ShortPassword() {
	env, err := s.setupEnv(false)
	s.Require().NoError(err)

	res := s.register(s.hostFor(env.ID), "pw@example.com", "short")

	s.Equal(http.StatusBadRequest, res.Status)
	s.Equal("password must be at least 8 characters", s.errors(res)[0])
}

func (s *WithSKAuthEmailSuite) Test_Register_AuthNotEnabled() {
	env := s.MockEnv(s.app, nil)

	res := s.register(s.hostFor(env.ID), "noauth@example.com", "supersecret123")

	s.Equal(http.StatusNotFound, res.Status)
}

func (s *WithSKAuthEmailSuite) Test_Register_EmailProviderNotEnabled() {
	env := s.MockEnv(s.app, map[string]any{
		"AuthConf": &buildconf.SKAuthConf{
			Secret: "test-secret-padded-to-32-chars!!",
			Status: true,
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

	res := s.register(s.hostFor(env.ID), "noprovider@example.com", "supersecret123")

	s.Equal(http.StatusNotFound, res.Status)
}

func (s *WithSKAuthEmailSuite) Test_Register_WithMailer_RequiresVerification() {
	env, err := s.setupEnv(true)
	s.Require().NoError(err)

	res := s.register(s.hostFor(env.ID), "mailer@example.com", "supersecret123")

	s.Equal(http.StatusCreated, res.Status)

	data := s.jsonData(res)
	s.Equal(true, data["verificationRequired"])
	s.Equal("mailer@example.com", data["email"])
	s.NotEmpty(data["userId"])
	s.Empty(data["token"])
}

// Login tests

func (s *WithSKAuthEmailSuite) Test_Login_Success() {
	env, err := s.setupEnv(false)
	s.Require().NoError(err)

	host := s.hostFor(env.ID)

	s.Require().Equal(http.StatusCreated, s.register(host, "login@example.com", "supersecret123").Status)

	res := s.login(host, "login@example.com", "supersecret123")

	s.Equal(http.StatusOK, res.Status)

	data := s.jsonData(res)
	s.NotEmpty(data["token"])
	s.Equal("login@example.com", data["email"])
	s.NotEmpty(data["userId"])
}

// Test_Login_CookieMode_SetsCookieNoToken verifies that in cookie mode login sets
// the session cookie and omits the token from the body, so the browser holds a
// single credential (no localStorage-vs-cookie drift).
func (s *WithSKAuthEmailSuite) Test_Login_CookieMode_SetsCookieNoToken() {
	env, err := s.setupEnv(false)
	s.Require().NoError(err)

	host := s.hostForCookie(env.ID)
	origin := s.sameHostOrigin(host)

	s.Require().Equal(http.StatusCreated, s.registerOrigin(host, origin, "cookie@example.com", "supersecret123").Status)

	res := s.loginOrigin(host, origin, "cookie@example.com", "supersecret123")

	s.Equal(http.StatusOK, res.Status)

	data := s.jsonData(res)
	_, hasToken := data["token"]
	s.False(hasToken, "cookie mode must not return the token in the body")
	s.Equal("cookie@example.com", data["email"])

	s.Require().Len(res.Cookies, 1)
	s.Equal(buildconf.SessionCookieName, res.Cookies[0].Name)
	s.Equal(".example.com", res.Cookies[0].Domain)
	s.True(res.Cookies[0].HttpOnly)
	s.True(res.Cookies[0].Secure)
	s.Equal(http.SameSiteLaxMode, res.Cookies[0].SameSite)
}

// Test_Login_CookieMode_BearerDelivery verifies that a native/mobile client on a
// cookie-mode env can opt into bearer delivery: with X-Session-Delivery: bearer
// (and no Origin, as native clients send) it gets the token in the body and no
// cookie, and is not blocked by the cookie-CSRF Origin guard.
func (s *WithSKAuthEmailSuite) Test_Login_CookieMode_BearerDelivery() {
	env, err := s.setupEnv(false)
	s.Require().NoError(err)

	host := s.hostForCookie(env.ID)

	s.Require().Equal(http.StatusCreated, s.registerOrigin(host, s.sameHostOrigin(host), "mobile@example.com", "supersecret123").Status)

	res := s.loginBearer(host, "mobile@example.com", "supersecret123")

	s.Equal(http.StatusOK, res.Status)

	data := s.jsonData(res)
	s.NotEmpty(data["token"], "bearer delivery must return the token in the body")
	s.Empty(res.Cookies, "bearer delivery must not set a cookie")
}

// Test_Register_CookieMode_SetsCookieNoToken verifies the same single-credential
// behavior on the auto-login register path (no mailer / verification).
func (s *WithSKAuthEmailSuite) Test_Register_CookieMode_SetsCookieNoToken() {
	env, err := s.setupEnv(false)
	s.Require().NoError(err)

	host := s.hostForCookie(env.ID)

	res := s.registerOrigin(host, s.sameHostOrigin(host), "cookiereg@example.com", "supersecret123")

	s.Equal(http.StatusCreated, res.Status)

	data := s.jsonData(res)
	_, hasToken := data["token"]
	s.False(hasToken, "cookie mode must not return the token in the body")

	s.Require().Len(res.Cookies, 1)
	s.Equal(buildconf.SessionCookieName, res.Cookies[0].Name)
}

// Test_Login_CookieMode_ForeignOrigin_Rejected verifies the login-CSRF guard: in
// cookie mode a login POST carrying a foreign (non-allow-listed) Origin is
// refused with 403 and no session cookie, so an attacker can't plant their own
// session on the victim's browser via a cross-site form post.
func (s *WithSKAuthEmailSuite) Test_Login_CookieMode_ForeignOrigin_Rejected() {
	env, err := s.setupEnv(false)
	s.Require().NoError(err)

	host := s.hostForCookie(env.ID)
	origin := s.sameHostOrigin(host)

	s.Require().Equal(http.StatusCreated, s.registerOrigin(host, origin, "victim@example.com", "supersecret123").Status)

	res := s.loginOrigin(host, "https://evil.example.org", "victim@example.com", "supersecret123")

	s.Equal(http.StatusForbidden, res.Status)
	s.Empty(res.Cookies, "no session cookie may be set for a cross-origin login")
}

// Test_Login_CookieMode_MissingOrigin_Rejected verifies that a cookie-mode login
// with no Origin header (browsers always send one on POST) is treated as
// cross-site and refused, closing the CSRF vector for form posts.
func (s *WithSKAuthEmailSuite) Test_Login_CookieMode_MissingOrigin_Rejected() {
	env, err := s.setupEnv(false)
	s.Require().NoError(err)

	host := s.hostForCookie(env.ID)

	s.Require().Equal(http.StatusCreated, s.registerOrigin(host, s.sameHostOrigin(host), "victim2@example.com", "supersecret123").Status)

	// A browser cookie login with no Origin and no bearer opt-in: must be refused.
	res, err := hosting.ServeAuth(s.postRequest(host, "/_stormkit/auth/login", map[string]any{
		"email":    "victim2@example.com",
		"password": "supersecret123",
	}))
	s.Require().NoError(err)

	s.Equal(http.StatusForbidden, res.Status)
	s.Empty(res.Cookies, "no session cookie may be set when Origin is absent")
}

func (s *WithSKAuthEmailSuite) Test_Login_WrongPassword() {
	env, err := s.setupEnv(false)
	s.Require().NoError(err)

	host := s.hostFor(env.ID)

	s.Require().Equal(http.StatusCreated, s.register(host, "login2@example.com", "supersecret123").Status)

	res := s.login(host, "login2@example.com", "wrongpassword")

	s.Equal(http.StatusBadRequest, res.Status)
	s.Equal("invalid email or password", s.errors(res)[0])
}

func (s *WithSKAuthEmailSuite) Test_Login_UnknownEmail() {
	env, err := s.setupEnv(false)
	s.Require().NoError(err)

	res := s.login(s.hostFor(env.ID), "nobody@example.com", "supersecret123")

	s.Equal(http.StatusBadRequest, res.Status)
	s.Equal("invalid email or password", s.errors(res)[0])
}

func (s *WithSKAuthEmailSuite) Test_Login_InvalidEmail() {
	env, err := s.setupEnv(false)
	s.Require().NoError(err)

	res := s.login(s.hostFor(env.ID), "not-an-email", "supersecret123")

	s.Equal(http.StatusBadRequest, res.Status)
	s.Equal("email is invalid", s.errors(res)[0])
}

func (s *WithSKAuthEmailSuite) Test_Login_ShortPassword() {
	env, err := s.setupEnv(false)
	s.Require().NoError(err)

	res := s.login(s.hostFor(env.ID), "login3@example.com", "short")

	s.Equal(http.StatusBadRequest, res.Status)
	s.Equal("password must be at least 8 characters", s.errors(res)[0])
}

func (s *WithSKAuthEmailSuite) Test_Login_AuthNotEnabled() {
	env := s.MockEnv(s.app, nil)

	res := s.login(s.hostFor(env.ID), "login4@example.com", "supersecret123")

	s.Equal(http.StatusNotFound, res.Status)
}

func (s *WithSKAuthEmailSuite) Test_Login_EmailProviderNotEnabled() {
	env := s.MockEnv(s.app, map[string]any{
		"AuthConf": &buildconf.SKAuthConf{
			Secret: "test-secret-padded-to-32-chars!!",
			Status: true,
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

	res := s.login(s.hostFor(env.ID), "login5@example.com", "supersecret123")

	s.Equal(http.StatusNotFound, res.Status)
}

func (s *WithSKAuthEmailSuite) Test_Login_UnverifiedUser() {
	env, err := s.setupEnv(true)
	s.Require().NoError(err)

	host := s.hostFor(env.ID)

	s.Require().Equal(http.StatusCreated, s.register(host, "unverified@example.com", "supersecret123").Status)

	res := s.login(host, "unverified@example.com", "supersecret123")

	s.Equal(http.StatusForbidden, res.Status)
	s.Equal("please verify your email address before logging in", s.errors(res)[0])
}

// Verify tests

func (s *WithSKAuthEmailSuite) Test_Verify_Success() {
	env, err := s.setupEnv(true)
	s.Require().NoError(err)

	host := s.hostFor(env.ID)
	var capturedToken string

	buildconf.SendMailFunc = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		body := string(msg)
		const marker = "?token="

		for i := 0; i <= len(body)-len(marker); i++ {
			if body[i:i+len(marker)] == marker {
				end := i + len(marker)

				for end < len(body) && body[end] != '"' && body[end] != '&' && body[end] != ' ' {
					end++
				}

				capturedToken = body[i+len(marker) : end]
				break
			}
		}

		return nil
	}

	s.Require().Equal(http.StatusCreated, s.register(host, "willverify@example.com", "supersecret123").Status)
	s.Require().NotEmpty(capturedToken)

	res := hosting.HandleAuthVerify(s.getRequest(host, fmt.Sprintf("/_stormkit/auth/verify?token=%s&envId=%d", capturedToken, env.ID)))

	// Verification sets the session cookie and 302s to the success URL.
	s.Equal(http.StatusFound, res.Status)
	s.Require().Len(res.Cookies, 1)
	s.Equal(buildconf.SessionCookieName, res.Cookies[0].Name)
}

func (s *WithSKAuthEmailSuite) Test_Verify_InvalidToken() {
	env, err := s.setupEnv(true)
	s.Require().NoError(err)

	res := hosting.HandleAuthVerify(s.getRequest(s.hostFor(env.ID), fmt.Sprintf("/_stormkit/auth/verify?token=not-a-real-token&envId=%d", env.ID)))

	s.Equal(http.StatusBadRequest, res.Status)
	s.Contains(string(res.Data.([]byte)), "invalid or expired verification token")
}

func (s *WithSKAuthEmailSuite) Test_Verify_AfterVerification_LoginSucceeds() {
	env, err := s.setupEnv(true)
	s.Require().NoError(err)

	host := s.hostFor(env.ID)
	var capturedToken string

	buildconf.SendMailFunc = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		body := string(msg)
		const marker = "?token="

		for i := 0; i <= len(body)-len(marker); i++ {
			if body[i:i+len(marker)] == marker {
				end := i + len(marker)

				for end < len(body) && body[end] != '"' && body[end] != '&' && body[end] != ' ' {
					end++
				}

				capturedToken = body[i+len(marker) : end]
				break
			}
		}

		return nil
	}

	s.Require().Equal(http.StatusCreated, s.register(host, "loginafter@example.com", "supersecret123").Status)
	s.Require().NotEmpty(capturedToken)

	hosting.HandleAuthVerify(s.getRequest(host, fmt.Sprintf("/_stormkit/auth/verify?token=%s&envId=%d", capturedToken, env.ID)))

	res := s.login(host, "loginafter@example.com", "supersecret123")

	s.Equal(http.StatusOK, res.Status)
	s.NotEmpty(s.jsonData(res)["token"])
}

func TestWithSKAuthEmail(t *testing.T) {
	suite.Run(t, new(WithSKAuthEmailSuite))
}
