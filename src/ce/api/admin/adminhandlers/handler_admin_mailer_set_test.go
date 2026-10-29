package adminhandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type HandlerAdminMailerSetSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	usr  *factory.MockUser
}

func (s *HandlerAdminMailerSetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.usr = s.MockUser(map[string]any{"IsAdmin": true})
}

func (s *HandlerAdminMailerSetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetCache(context.Background())
}

func (s *HandlerAdminMailerSetSuite) post(usr *factory.MockUser, body map[string]any) shttptest.Response {
	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/mailer",
		body,
		map[string]string{"Authorization": usertest.Authorization(usr.ID)},
	)
}

func (s *HandlerAdminMailerSetSuite) valid() map[string]any {
	return map[string]any{
		"host":     "smtp.instance.test",
		"port":     "2525",
		"username": "instance-user",
		"password": "instance-secret",
		"from":     "Stormkit <noreply@instance.test>",
	}
}

// stored reads the mailer back from the database, not from the cache.
func (s *HandlerAdminMailerSetSuite) stored() *admin.MailerConfig {
	admin.ResetCache(context.Background())

	cnf, err := admin.Store().Config(context.Background())
	s.Require().NoError(err)

	return cnf.MailerConfig
}

func (s *HandlerAdminMailerSetSuite) Test_Success() {
	response := s.post(s.usr, s.valid())
	body := response.String()

	s.Equal(http.StatusOK, response.Code)
	s.NotContains(body, "instance-secret")
	s.Contains(body, admin.MailerPasswordPlaceholder)

	stored := s.stored()
	s.Require().NotNil(stored)
	s.Equal("smtp.instance.test", stored.Host)
	s.Equal("instance-secret", stored.Password)
	s.Equal("Stormkit <noreply@instance.test>", stored.From)
}

// Test_CredentialsEncrypted verifies that the SMTP username and password are
// not stored in plain text.
func (s *HandlerAdminMailerSetSuite) Test_CredentialsEncrypted() {
	s.Equal(http.StatusOK, s.post(s.usr, s.valid()).Code)

	var raw string
	s.Require().NoError(s.conn.QueryRow("SELECT config_data::text FROM stormkit_config WHERE config_id = 1").Scan(&raw))
	s.NotContains(raw, "instance-secret")
	s.NotContains(raw, "instance-user")
	s.Contains(raw, "smtp.instance.test")
}

func (s *HandlerAdminMailerSetSuite) Test_KeepsPasswordWhenOmitted() {
	s.Equal(http.StatusOK, s.post(s.usr, s.valid()).Code)

	body := s.valid()
	body["password"] = admin.MailerPasswordPlaceholder
	body["port"] = "587"

	s.Equal(http.StatusOK, s.post(s.usr, body).Code)
	s.Equal("instance-secret", s.stored().Password)
	s.Equal("587", s.stored().Port)
}

// Test_DropsPasswordWhenHostChanges verifies that the stored password cannot
// be sent to another server without re-entering it.
func (s *HandlerAdminMailerSetSuite) Test_DropsPasswordWhenHostChanges() {
	s.Equal(http.StatusOK, s.post(s.usr, s.valid()).Code)

	body := s.valid()
	body["host"] = "smtp.attacker.test"
	body["password"] = ""

	response := s.post(s.usr, body)

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(response.String(), "Password is a required field")
	s.Equal("smtp.instance.test", s.stored().Host)
}

func (s *HandlerAdminMailerSetSuite) Test_Invalid() {
	response := s.post(s.usr, map[string]any{"port": "99999", "from": "not an address"})
	body := response.String()

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(body, "SMTP Host is a required field")
	s.Contains(body, "Username is a required field")
	s.Contains(body, "Password is a required field")
	s.Contains(body, "From must be an email address")
	s.Contains(body, "SMTP Port must be a number")
	s.Nil(s.stored())
}

func (s *HandlerAdminMailerSetSuite) Test_NotAdmin() {
	response := s.post(s.MockUser(nil), s.valid())

	s.Equal(http.StatusUnauthorized, response.Code)
	s.Nil(s.stored())
}

func TestHandlerAdminMailerSetSuite(t *testing.T) {
	suite.Run(t, &HandlerAdminMailerSetSuite{})
}
