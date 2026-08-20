package mailerhandlers_test

import (
	"fmt"
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

type HandlerMailerConfigGetSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerMailerConfigGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerMailerConfigGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	buildconf.SendMailFunc = smtp.SendMail
}

func (s *HandlerMailerConfigGetSuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, map[string]any{
		"MailerConf": &buildconf.MailerConf{
			Host:     "smtp.gmail.com",
			Port:     "587",
			Username: "test",
			Password: "testpwd",
		},
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(mailerhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/mailer/config?appId=%d&envId=%d", app.ID, env.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{ 
		"config": { 
			"host": "smtp.gmail.com",
			"port": "587",
			"password": "****-****-****-****",
			"username": "test"
		}
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

// Test_MasksPassword is the guarantee the MCP tools rely on: the stored SMTP
// password never leaves the server, so an agent that can read the config
// cannot exfiltrate the credential.
func (s *HandlerMailerConfigGetSuite) Test_MasksPassword() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, map[string]any{
		"MailerConf": &buildconf.MailerConf{
			Host:     "smtp.gmail.com",
			Username: "test",
			Password: "super-secret",
		},
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(mailerhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/mailer/config?appId=%d&envId=%d", app.ID, env.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	body := response.String()

	s.Equal(http.StatusOK, response.Code)
	s.NotContains(body, "super-secret")
	s.Contains(body, buildconf.PasswordPlaceholder)
}

func (s *HandlerMailerConfigGetSuite) Test_NoConfig() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(mailerhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/mailer/config?appId=%d&envId=%d", app.ID, env.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{"config": null}`, response.String())
}

func (s *HandlerMailerConfigGetSuite) Test_UnknownEnvID() {
	usr := s.MockUser()
	app := s.MockApp(usr)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(mailerhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/mailer/config?appId=%d&envId=999999999", app.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusNotFound, response.Code)
}

func TestHandlerMailerConfigGetSuite(t *testing.T) {
	suite.Run(t, &HandlerMailerConfigGetSuite{})
}
