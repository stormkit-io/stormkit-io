package mailerhandlers_test

import (
	"context"
	"net/http"
	"net/smtp"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/mailerhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerMailerConfigSetSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerMailerConfigSetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerMailerConfigSetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	buildconf.SendMailFunc = smtp.SendMail
}

func (s *HandlerMailerConfigSetSuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(mailerhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/mailer/config",
		map[string]string{
			"appId":    app.ID.String(),
			"envId":    env.ID.String(),
			"smtpHost": "smtp.gmail.com",
			"smtpPort": "587",
			"username": "test-user",
			"password": "test-pwd",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	envUpdated, err := buildconf.NewStore().EnvironmentByID(context.Background(), env.ID)
	s.NoError(err)
	s.NotNil(envUpdated)
	s.Equal(&buildconf.MailerConf{
		Host:     "smtp.gmail.com",
		Port:     "587",
		Username: "test-user",
		Password: "test-pwd",
	}, envUpdated.MailerConf)
}

func (s *HandlerMailerConfigSetSuite) Test_BadRequest() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(mailerhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/mailer/config",
		map[string]string{
			"appId":    app.ID.String(),
			"envId":    env.ID.String(),
			"username": "test-user",
			"password": "test-pwd",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{
		"errors": {
			"host": "SMTP Host is a required field."
		}
	}`

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(expected, response.String())
}

// Test_KeepsPasswordWhenOmitted covers the round-trip a masked client makes:
// it reads the config, gets the placeholder, and posts it straight back.
func (s *HandlerMailerConfigSetSuite) Test_KeepsPasswordWhenOmitted() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, map[string]any{
		"MailerConf": &buildconf.MailerConf{
			Host:     "smtp.gmail.com",
			Port:     "587",
			Username: "test-user",
			Password: "original-pwd",
		},
	})

	for _, password := range []string{"", buildconf.PasswordPlaceholder} {
		response := shttptest.RequestWithHeaders(
			shttp.NewRouter().RegisterService(mailerhandlers.Services).Router().Handler(),
			shttp.MethodPost,
			"/mailer/config",
			map[string]string{
				"appId":    app.ID.String(),
				"envId":    env.ID.String(),
				"smtpHost": "smtp.gmail.com",
				"password": password,
			},
			map[string]string{
				"Authorization": usertest.Authorization(usr.ID),
			},
		)

		s.Equal(http.StatusOK, response.Code)

		envUpdated, err := buildconf.NewStore().EnvironmentByID(context.Background(), env.ID)
		s.NoError(err)
		s.Equal("original-pwd", envUpdated.MailerConf.Password)
		s.Equal("smtp.gmail.com", envUpdated.MailerConf.Host)
		s.Equal("test-user", envUpdated.MailerConf.Username)
	}
}

// Test_ClearsPasswordWhenHostChanges closes the exfiltration path the masking
// would otherwise leave open: repointing the config at a server the caller
// controls while the stored credential is retained makes the next send hand
// that credential over.
func (s *HandlerMailerConfigSetSuite) Test_ClearsPasswordWhenHostChanges() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, map[string]any{
		"MailerConf": &buildconf.MailerConf{
			Host:     "smtp.gmail.com",
			Port:     "587",
			Username: "test-user",
			Password: "original-pwd",
		},
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(mailerhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/mailer/config",
		map[string]string{
			"appId":    app.ID.String(),
			"envId":    env.ID.String(),
			"smtpHost": "smtp.attacker.tld",
			"password": buildconf.PasswordPlaceholder,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(response.String(), "Password is a required field.")

	envUpdated, err := buildconf.NewStore().EnvironmentByID(context.Background(), env.ID)
	s.NoError(err)
	s.Equal("smtp.gmail.com", envUpdated.MailerConf.Host)
	s.Equal("original-pwd", envUpdated.MailerConf.Password)
}

func (s *HandlerMailerConfigSetSuite) Test_NeverEchoesPassword() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(mailerhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/mailer/config",
		map[string]string{
			"appId":    app.ID.String(),
			"envId":    env.ID.String(),
			"smtpHost": "smtp.gmail.com",
			"username": "test-user",
			"password": "super-secret",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.NotContains(response.String(), "super-secret")
}

func (s *HandlerMailerConfigSetSuite) Test_InvalidPort() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(mailerhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/mailer/config",
		map[string]string{
			"appId":    app.ID.String(),
			"envId":    env.ID.String(),
			"smtpHost": "smtp.gmail.com",
			"smtpPort": "not-a-port",
			"username": "test-user",
			"password": "test-pwd",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(response.String(), "SMTP Port must be a number")
}

// Test_UnknownEnvID guards a nil dereference: WithApp only asserts that an
// envId was provided, and the store returns (nil, nil) for an unknown id.
func (s *HandlerMailerConfigSetSuite) Test_UnknownEnvID() {
	usr := s.MockUser()
	app := s.MockApp(usr)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(mailerhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/mailer/config",
		map[string]string{
			"appId":    app.ID.String(),
			"envId":    "999999999",
			"smtpHost": "smtp.gmail.com",
			"username": "test-user",
			"password": "test-pwd",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusNotFound, response.Code)
}

func TestHandlerMailerConfigSetSuite(t *testing.T) {
	suite.Run(t, &HandlerMailerConfigSetSuite{})
}
