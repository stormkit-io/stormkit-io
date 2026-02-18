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
	mailerhandlers.SendMail = smtp.SendMail
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

func TestHandlerMailerConfigSetSuite(t *testing.T) {
	suite.Run(t, &HandlerMailerConfigSetSuite{})
}
