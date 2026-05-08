package publicapiv1_test

import (
	"context"
	"fmt"
	"net/http"
	"net/smtp"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthVerifySuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	app  *factory.MockApp
}

func (s *HandlerAuthVerifySuite) BeforeTest(suiteName, _ string) {
	s.truncateAuthTables()
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.app = s.MockApp(s.MockUser(nil), nil)
	config.SetIsSelfHosted(true)
	buildconf.SendMailFunc = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		return nil
	}
}

func (s *HandlerAuthVerifySuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	config.SetIsSelfHosted(false)
	buildconf.SendMailFunc = smtp.SendMail
}

func (s *HandlerAuthVerifySuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

func (s *HandlerAuthVerifySuite) truncateAuthTables() {
	truncateAuthTables()
}

func (s *HandlerAuthVerifySuite) setupEnv(withMailer bool) (*factory.MockEnv, error) {
	fields := map[string]any{
		"AuthConf": &buildconf.SKAuthConf{
			Secret:     "test-secret",
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

func (s *HandlerAuthVerifySuite) register(envID any, email, password string) shttptest.Response {
	return shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/auth/register",
		map[string]string{
			"envId":    fmt.Sprintf("%v", envID),
			"email":    email,
			"password": password,
		},
		nil,
	)
}

func (s *HandlerAuthVerifySuite) verify(token, envID any) shttptest.Response {
	return shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/auth/verify?token=%v&envId=%v", token, envID),
		nil,
		nil,
	)
}

// Test_Register_WithMailer checks that registration with a configured mailer does not
// return a session token, and instead signals that verification is required.
func (s *HandlerAuthVerifySuite) Test_Register_WithMailer() {
	env, err := s.setupEnv(true)
	s.Require().NoError(err)

	resp := s.register(env.ID, "verify@example.com", "supersecret123")

	s.Equal(http.StatusCreated, resp.Code)

	body := resp.Map()
	s.Equal(true, body["verificationRequired"])
	s.Equal("verify@example.com", body["email"])
	s.NotEmpty(body["userId"])
	s.Empty(body["token"])
}

// Test_Register_WithoutMailer checks that registration without a mailer still returns
// a session token immediately (unchanged behavior).
func (s *HandlerAuthVerifySuite) Test_Register_WithoutMailer() {
	env, err := s.setupEnv(false)
	s.Require().NoError(err)

	resp := s.register(env.ID, "noverify@example.com", "supersecret123")

	s.Equal(http.StatusCreated, resp.Code)
	s.NotEmpty(resp.Map()["token"])
}

// Test_Login_UnverifiedUser checks that login is blocked when the user has not verified
// their email and the env has a mailer configured.
func (s *HandlerAuthVerifySuite) Test_Login_UnverifiedUser() {
	env, err := s.setupEnv(true)
	s.Require().NoError(err)

	s.Require().Equal(http.StatusCreated, s.register(env.ID, "unverified@example.com", "supersecret123").Code)

	resp := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/auth/login",
		map[string]string{
			"envId":    fmt.Sprintf("%d", env.ID),
			"email":    "unverified@example.com",
			"password": "supersecret123",
		},
		nil,
	)

	s.Equal(http.StatusForbidden, resp.Code)
	s.Equal("please verify your email address before logging in", resp.Map()["errors"].([]any)[0])
}

// Test_Verify_Success checks the full happy path: register → capture token → verify → get session.
func (s *HandlerAuthVerifySuite) Test_Verify_Success() {
	env, err := s.setupEnv(true)
	s.Require().NoError(err)

	var capturedToken string

	buildconf.SendMailFunc = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		// The email body contains the token as a query parameter; extract it.
		body := string(msg)
		const marker = "?token="
		idx := -1

		for i := 0; i <= len(body)-len(marker); i++ {
			if body[i:i+len(marker)] == marker {
				idx = i + len(marker)
				break
			}
		}

		if idx >= 0 {
			end := idx
			for end < len(body) && body[end] != '"' && body[end] != '&' && body[end] != ' ' {
				end++
			}
			capturedToken = body[idx:end]
		}

		return nil
	}

	s.Require().Equal(http.StatusCreated, s.register(env.ID, "willverify@example.com", "supersecret123").Code)
	s.Require().NotEmpty(capturedToken)

	resp := s.verify(capturedToken, env.ID)

	s.Equal(http.StatusOK, resp.Code)

	body := resp.Map()
	s.NotEmpty(body["token"])
	s.Equal("willverify@example.com", body["email"])
	s.NotEmpty(body["userId"])
}

// Test_Verify_InvalidToken checks that an unknown token returns 400.
func (s *HandlerAuthVerifySuite) Test_Verify_InvalidToken() {
	env, err := s.setupEnv(true)
	s.Require().NoError(err)

	resp := s.verify("not-a-real-token", env.ID)

	s.Equal(http.StatusBadRequest, resp.Code)
	s.Equal("invalid or expired verification token", resp.Map()["errors"].([]any)[0])
}

// Test_Verify_MissingToken checks that omitting the token returns 400.
func (s *HandlerAuthVerifySuite) Test_Verify_MissingToken() {
	resp := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		"/v1/auth/verify?envId=1",
		nil,
		nil,
	)

	s.Equal(http.StatusBadRequest, resp.Code)
	s.Equal("token is required", resp.Map()["errors"].([]any)[0])
}

// Test_Verify_AfterVerification_LoginSucceeds checks that a verified user can log in.
func (s *HandlerAuthVerifySuite) Test_Verify_AfterVerification_LoginSucceeds() {
	env, err := s.setupEnv(true)
	s.Require().NoError(err)

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

	s.Require().Equal(http.StatusCreated, s.register(env.ID, "loginafter@example.com", "supersecret123").Code)
	s.Require().Equal(http.StatusOK, s.verify(capturedToken, env.ID).Code)

	resp := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/auth/login",
		map[string]string{
			"envId":    fmt.Sprintf("%d", env.ID),
			"email":    "loginafter@example.com",
			"password": "supersecret123",
		},
		nil,
	)

	s.Equal(http.StatusOK, resp.Code)
	s.NotEmpty(resp.Map()["token"])
}

func TestHandlerAuthVerifySuite(t *testing.T) {
	suite.Run(t, &HandlerAuthVerifySuite{})
}
