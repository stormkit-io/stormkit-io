package mailerhandlers_test

import (
	"context"
	"fmt"
	"net/http"
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

type MailListSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *MailListSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *MailListSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *MailListSuite) Test_ReturnsEmails() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	err := buildconf.MailerStore().InsertEmail(context.Background(), buildconf.Email{
		EnvID:   env.ID,
		To:      "joe@stormkit.io",
		From:    "sender@example.com",
		Subject: "Hello",
		Body:    "Welcome!",
	})
	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(mailerhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/mailer?appId=%d&envId=%d", app.ID, env.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	emails, err := buildconf.MailerStore().Emails(context.Background(), env.ID)
	s.NoError(err)
	s.Len(emails, 1)
	s.Equal("Hello", emails[0].Subject)
	s.Equal("Welcome!", emails[0].Body)
}

func (s *MailListSuite) Test_EmptyList() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(mailerhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/mailer?appId=%d&envId=%d", app.ID, env.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{"emails": []}`, response.String())
}

func TestMailListSuite(t *testing.T) {
	suite.Run(t, &MailListSuite{})
}
