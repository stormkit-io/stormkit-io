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

type MailerSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *MailerSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *MailerSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	mailerhandlers.SendMail = smtp.SendMail
}

func (s *MailerSuite) Test_Success() {
	called := false
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, map[string]any{
		"MailerConf": &buildconf.MailerConf{
			Host:     "smtp.gmail.com",
			Port:     "587",
			Username: "test",
			Password: "test",
		},
	})

	mailerhandlers.SendMail = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
		s.Equal("smtp.gmail.com:587", addr)
		s.Equal("test", from)
		s.Equal([]string{"joe@stormkit.io", "jane@stormkit.io"}, to)
		s.Equal([]byte(
			""+
				"From: Stormkit <hello@stormkit.io>\n"+
				"Subject: Hello\n"+mime+
				"<html><body>Welcome to my world!</body></html>"), msg)
		called = true
		return nil
	}

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(mailerhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/mailer",
		map[string]any{
			"appId":   app.ID.String(),
			"envId":   env.ID.String(),
			"to":      "joe@stormkit.io; jane@stormkit.io",
			"from":    "Stormkit <hello@stormkit.io>",
			"subject": "Hello",
			"body":    "Welcome to my world!",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.True(called)

	emails, err := buildconf.MailerStore().Emails(context.Background(), env.ID)
	s.NoError(err)
	s.Len(emails, 1)
}

func (s *MailerSuite) Test_ShouldFail_NoConfig() {
	called := false
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	mailerhandlers.SendMail = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		called = true
		return nil
	}

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(mailerhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/mailer",
		map[string]any{
			"appId":   app.ID.String(),
			"envId":   env.ID.String(),
			"to":      "joe@stormkit.io; jane@stormkit.io",
			"subject": "Hello",
			"body":    "Welcome to my world!",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{ "error": "Mailer is not yet configured." }`, response.String())
	s.False(called)
}

func TestMailerSuite(t *testing.T) {
	suite.Run(t, &MailerSuite{})
}
