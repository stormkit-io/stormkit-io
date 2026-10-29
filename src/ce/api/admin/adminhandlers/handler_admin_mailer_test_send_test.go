package adminhandlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/smtp"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type HandlerAdminMailerTestSendSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerAdminMailerTestSendSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerAdminMailerTestSendSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetCache(context.Background())
	buildconf.SendMailFunc = buildconf.SendMailWithDeadline
}

func (s *HandlerAdminMailerTestSendSuite) configure() {
	cnf := admin.MustConfig().Clone()
	cnf.MailerConfig = &admin.MailerConfig{
		Host:     "smtp.instance.test",
		Username: "instance-user",
		Password: "instance-secret",
		From:     "Stormkit <noreply@instance.test>",
	}

	s.Require().NoError(admin.Store().UpsertConfig(context.Background(), cnf))
}

func (s *HandlerAdminMailerTestSendSuite) post(to string) shttptest.Response {
	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/mailer/test",
		map[string]string{"to": to},
		map[string]string{"Authorization": usertest.Authorization(s.MockUser(map[string]any{"IsAdmin": true}).ID)},
	)
}

func (s *HandlerAdminMailerTestSendSuite) Test_Success() {
	s.configure()

	var sentAddr, sentFrom string
	var sentTo []string

	buildconf.SendMailFunc = func(addr string, _ smtp.Auth, from string, to []string, _ []byte) error {
		sentAddr, sentFrom, sentTo = addr, from, to
		return nil
	}

	s.Equal(http.StatusOK, s.post("admin@instance.test").Code)
	s.Equal("smtp.instance.test:587", sentAddr)
	s.Equal("noreply@instance.test", sentFrom)
	s.Equal([]string{"admin@instance.test"}, sentTo)
}

func (s *HandlerAdminMailerTestSendSuite) Test_NotConfigured() {
	response := s.post("admin@instance.test")

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(response.String(), "Configure the default mailer first")
}

// Test_ReturnsSMTPError verifies that the admin sees why the SMTP server
// refused the email.
func (s *HandlerAdminMailerTestSendSuite) Test_ReturnsSMTPError() {
	s.configure()

	buildconf.SendMailFunc = func(string, smtp.Auth, string, []string, []byte) error {
		return errors.New("535 Authentication Credentials Invalid")
	}

	response := s.post("admin@instance.test")

	s.Equal(http.StatusBadGateway, response.Code)
	s.Contains(response.String(), "535 Authentication Credentials Invalid")
}

func (s *HandlerAdminMailerTestSendSuite) Test_InvalidRecipient() {
	s.configure()

	s.Equal(http.StatusBadRequest, s.post("not-an-email").Code)
}

func TestHandlerAdminMailerTestSendSuite(t *testing.T) {
	suite.Run(t, &HandlerAdminMailerTestSendSuite{})
}
