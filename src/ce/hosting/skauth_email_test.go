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
	buildconf.SendMailFunc = smtp.SendMail
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

func (s *WithSKAuthEmailSuite) register(host *hosting.Host, email, password string) *shttp.Response {
	res, err := hosting.ServeAuth(s.postRequest(host, "/_stormkit/auth/register", map[string]any{
		"email":    email,
		"password": password,
	}))
	s.Require().NoError(err)
	return res
}

func (s *WithSKAuthEmailSuite) login(host *hosting.Host, email, password string) *shttp.Response {
	res, err := hosting.ServeAuth(s.postRequest(host, "/_stormkit/auth/login", map[string]any{
		"email":    email,
		"password": password,
	}))
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
	res, err := hosting.ServeAuth(s.postRequest(s.hostFor(env.ID), "/_stormkit/auth/register", map[string]any{
		"envId":    "9999999",
		"email":    "crossenv@example.com",
		"password": "supersecret123",
	}))
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

	s.Equal(http.StatusOK, res.Status)

	body := string(res.Data.([]byte))
	s.Contains(body, "localStorage.setItem('skauth'")
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
