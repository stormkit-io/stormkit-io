package publicapiv1_test

import (
	"context"
	"fmt"
	"net/http"
	"net/smtp"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerMailerSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	usr  *factory.MockUser
	app  *factory.MockApp
}

func (s *HandlerMailerSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.usr = s.MockUser(nil)
	s.app = s.MockApp(s.usr, nil)
}

func (s *HandlerMailerSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	buildconf.SendMailFunc = buildconf.SendMailWithDeadline
}

func (s *HandlerMailerSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

func (s *HandlerMailerSuite) auth() map[string]string {
	return map[string]string{"Authorization": usertest.Authorization(s.usr.ID)}
}

func (s *HandlerMailerSuite) configuredEnv() *factory.MockEnv {
	return s.MockEnv(s.app, map[string]any{
		"MailerConf": &buildconf.MailerConf{
			Host:     "smtp.gmail.com",
			Port:     "587",
			Username: "noreply@acme.com",
			Password: "super-secret",
		},
	})
}

func (s *HandlerMailerSuite) Test_ConfigGet_MasksPassword() {
	env := s.configuredEnv()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/mailer/config?appId=%d&envId=%d", s.app.ID, env.ID),
		nil,
		s.auth(),
	)

	body := response.String()

	s.Equal(http.StatusOK, response.Code)
	s.NotContains(body, "super-secret")
	s.Contains(body, buildconf.PasswordPlaceholder)
}

func (s *HandlerMailerSuite) Test_ConfigSet_KeepsPasswordAndNeverEchoes() {
	env := s.configuredEnv()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/mailer/config",
		map[string]string{
			"appId":    s.app.ID.String(),
			"envId":    env.ID.String(),
			"smtpHost": "smtp.gmail.com",
			"smtpPort": "2525",
			"password": buildconf.PasswordPlaceholder,
		},
		s.auth(),
	)

	s.Equal(http.StatusOK, response.Code)
	s.NotContains(response.String(), "super-secret")

	stored, err := buildconf.NewStore().EnvironmentByID(context.Background(), env.ID)
	s.Require().NoError(err)
	s.Equal("super-secret", stored.MailerConf.Password)
	s.Equal("2525", stored.MailerConf.Port)
	s.Equal("smtp.gmail.com", stored.MailerConf.Host)
	s.Equal("noreply@acme.com", stored.MailerConf.Username)
}

// Test_ConfigSet_ClearsPasswordWhenHostChanges pins the credential-rebinding guard:
// retaining a password the caller may never read is only safe while it keeps
// pointing at the same account, so repointing the host requires a new one.
func (s *HandlerMailerSuite) Test_ConfigSet_ClearsPasswordWhenHostChanges() {
	env := s.configuredEnv()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/mailer/config",
		map[string]string{
			"appId":    s.app.ID.String(),
			"envId":    env.ID.String(),
			"smtpHost": "smtp.attacker.tld",
			"password": buildconf.PasswordPlaceholder,
		},
		s.auth(),
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(response.String(), "Password is a required field.")

	stored, err := buildconf.NewStore().EnvironmentByID(context.Background(), env.ID)
	s.Require().NoError(err)
	s.Equal("smtp.gmail.com", stored.MailerConf.Host)
	s.Equal("super-secret", stored.MailerConf.Password)
}

func (s *HandlerMailerSuite) Test_ConfigSet_Invalid() {
	env := s.MockEnv(s.app, nil)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/mailer/config",
		map[string]string{
			"appId":    s.app.ID.String(),
			"envId":    env.ID.String(),
			"username": "noreply@acme.com",
		},
		s.auth(),
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(response.String(), "SMTP Host is a required field.")
}

func (s *HandlerMailerSuite) Test_Send() {
	env := s.configuredEnv()
	sentTo := []string{}

	buildconf.SendMailFunc = func(_ string, _ smtp.Auth, _ string, to []string, _ []byte) error {
		sentTo = to
		return nil
	}

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/mail",
		map[string]string{
			"appId":   s.app.ID.String(),
			"envId":   env.ID.String(),
			"to":      "someone@acme.com",
			"subject": "Test",
			"body":    "Hello",
		},
		s.auth(),
	)

	s.Equal(http.StatusOK, response.Code)
	s.Equal([]string{"someone@acme.com"}, sentTo)
	s.Contains(response.String(), `"delivered":true`)
}

// Test_Send_ReportsUndelivered pins the contract the dashboard relies on: an
// environment with no SMTP config still records the email, so a bare 200 would
// be a false positive on the button used to verify the mailer works.
func (s *HandlerMailerSuite) Test_Send_ReportsUndelivered() {
	env := s.MockEnv(s.app, nil)
	called := false

	buildconf.SendMailFunc = func(_ string, _ smtp.Auth, _ string, _ []string, _ []byte) error {
		called = true
		return nil
	}

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/mail",
		map[string]string{
			"appId":   s.app.ID.String(),
			"envId":   env.ID.String(),
			"to":      "someone@acme.com",
			"subject": "Test",
			"body":    "Hello",
		},
		s.auth(),
	)

	s.Equal(http.StatusOK, response.Code)
	s.Contains(response.String(), `"delivered":false`)
	s.False(called, "no SMTP server is configured, so nothing should be dialled")

	emails, err := buildconf.MailerStore().Emails(context.Background(), env.ID)
	s.Require().NoError(err)
	s.Len(emails, 1, "the email is still recorded")
}

func (s *HandlerMailerSuite) Test_Send_RequiresSubject() {
	env := s.configuredEnv()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/mail",
		map[string]string{
			"appId": s.app.ID.String(),
			"envId": env.ID.String(),
			"to":    "someone@acme.com",
			"body":  "Hello",
		},
		s.auth(),
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"errors":{"subject":"Subject is a required field."}}`, response.String())
}

// Test_EmailsGet_OmitsBody is the guarantee that keeps magic-link tokens out of
// the public API: the mailer log stores those emails verbatim, so a body would
// hand the caller a live sign-in link.
func (s *HandlerMailerSuite) Test_EmailsGet_OmitsBody() {
	env := s.MockEnv(s.app, nil)

	err := buildconf.MailerStore().InsertEmail(context.Background(), buildconf.Email{
		EnvID:   env.ID,
		To:      "someone@acme.com",
		From:    "noreply@acme.com",
		Subject: "Your magic link",
		Body:    `<a href="https://app.example.com/_stormkit/auth/magic?token=live-token">link</a>`,
	})

	s.Require().NoError(err)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/mailer/emails?appId=%d&envId=%d", s.app.ID, env.ID),
		nil,
		s.auth(),
	)

	body := response.String()

	s.Equal(http.StatusOK, response.Code)
	s.NotContains(body, "live-token")
	s.NotContains(body, "_stormkit/auth/magic")
	s.Contains(body, "Your magic link")

	// The recipient is the customer's end user: the domain survives so a
	// caller can confirm delivery, the local part does not.
	s.NotContains(body, "someone@acme.com")
	s.Contains(body, "s***@acme.com")
}

func (s *HandlerMailerSuite) Test_Forbidden_ForNonMember() {
	other := s.MockUser(nil)
	otherApp := s.MockApp(other, nil)
	env := s.MockEnv(otherApp, nil)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/mailer/config?appId=%d&envId=%d", otherApp.ID, env.ID),
		nil,
		s.auth(),
	)

	s.Equal(http.StatusForbidden, response.Code)
}

// Test_ConfigSet_AuditsPasswordRotation covers the one mailer write that
// changes nothing else: Insert() drops a diff whose Old and New are equal, so
// without an explicit marker replacing the sending credential would leave no
// audit trail at all.
func (s *HandlerMailerSuite) Test_ConfigSet_AuditsPasswordRotation() {
	admin.SetMockLicense()
	defer admin.ResetMockLicense()

	env := s.configuredEnv()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/mailer/config",
		map[string]string{
			"appId":    s.app.ID.String(),
			"envId":    env.ID.String(),
			"password": "rotated-secret",
		},
		s.auth(),
	)

	s.Equal(http.StatusOK, response.Code)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		EnvID: env.ID,
	})

	s.Require().NoError(err)
	s.Require().Len(audits, 1, "a credential rotation must be audited")
	s.Equal("UPDATE:MAILER", audits[0].Action)

	// The marker records that the credential changed, never the credential.
	s.Contains(response.String(), buildconf.PasswordPlaceholder)
	s.NotContains(response.String(), "rotated-secret")
}

// Test_Send_RecordsEvenWhenSendFails covers the ambiguous case: a relay can
// reject after it has already accepted the message, so a failed send must
// still leave a trace of what was attempted.
func (s *HandlerMailerSuite) Test_Send_RecordsEvenWhenSendFails() {
	env := s.configuredEnv()

	buildconf.SendMailFunc = func(_ string, _ smtp.Auth, _ string, _ []string, _ []byte) error {
		return fmt.Errorf("dial tcp 10.0.0.1:587: i/o timeout")
	}

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/mail",
		map[string]string{
			"appId":   s.app.ID.String(),
			"envId":   env.ID.String(),
			"to":      "someone@acme.com",
			"subject": "Test",
			"body":    "Hello",
		},
		s.auth(),
	)

	s.Equal(http.StatusInternalServerError, response.Code)

	// The SMTP host is caller-supplied, so the raw error must not come back.
	s.NotContains(response.String(), "10.0.0.1")

	emails, err := buildconf.MailerStore().Emails(context.Background(), env.ID)
	s.Require().NoError(err)
	s.Len(emails, 1, "a failed send is still recorded")
}

// Test_ConfigSet_InvalidPort covers the only validation rule on the mailer
// config that no other suite exercises.
func (s *HandlerMailerSuite) Test_ConfigSet_InvalidPort() {
	env := s.configuredEnv()

	for _, port := range []string{"not-a-port", "0", "99999"} {
		response := shttptest.RequestWithHeaders(
			s.handler(),
			shttp.MethodPost,
			"/v1/mailer/config",
			map[string]string{
				"appId":    s.app.ID.String(),
				"envId":    env.ID.String(),
				"smtpPort": port,
			},
			s.auth(),
		)

		s.Equal(http.StatusBadRequest, response.Code, port)
		s.Contains(response.String(), "SMTP Port must be a number between 1 and 65535.", port)
	}
}

// Test_ConfigSet_KeepsPasswordWhenBlank pins the dashboard's default save path:
// the password field renders empty, so an untouched form posts "". Treating
// that as a new value would wipe the stored credential on every save.
func (s *HandlerMailerSuite) Test_ConfigSet_KeepsPasswordWhenBlank() {
	env := s.configuredEnv()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/mailer/config",
		map[string]string{
			"appId":    s.app.ID.String(),
			"envId":    env.ID.String(),
			"smtpPort": "2525",
			"password": "",
		},
		s.auth(),
	)

	s.Equal(http.StatusOK, response.Code)

	stored, err := buildconf.NewStore().EnvironmentByID(context.Background(), env.ID)
	s.Require().NoError(err)
	s.Equal("super-secret", stored.MailerConf.Password, "a blank password keeps the stored one")
	s.Equal("2525", stored.MailerConf.Port)
}

// Test_ConfigGet_NoConfig covers an environment that never had a mailer - the
// first thing the dashboard requests when the tab is opened.
func (s *HandlerMailerSuite) Test_ConfigGet_NoConfig() {
	env := s.MockEnv(s.app, nil)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/mailer/config?appId=%s&envId=%s", s.app.ID, env.ID),
		nil,
		s.auth(),
	)

	s.Equal(http.StatusOK, response.Code)
	s.Contains(response.String(), `"config":null`)
}

// Test_Send_SplitsRecipients pins the semicolon-separated recipient list the
// tool description advertises, including the surrounding whitespace a caller
// naturally leaves in.
func (s *HandlerMailerSuite) Test_Send_SplitsRecipients() {
	env := s.configuredEnv()
	sentTo := []string{}
	sentAddr := ""

	buildconf.SendMailFunc = func(addr string, _ smtp.Auth, _ string, to []string, _ []byte) error {
		sentAddr, sentTo = addr, to
		return nil
	}

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/mail",
		map[string]string{
			"appId":   s.app.ID.String(),
			"envId":   env.ID.String(),
			"to":      "joe@acme.com; jane@acme.com",
			"subject": "Test",
			"body":    "Hello",
		},
		s.auth(),
	)

	s.Equal(http.StatusOK, response.Code)
	s.Equal([]string{"joe@acme.com", "jane@acme.com"}, sentTo)
	s.Equal("smtp.gmail.com:587", sentAddr)
}

// Test_Send_RecordsFields asserts the stored row field by field: a mapping
// regression in SendAndRecord would still leave exactly one email behind, so a
// length check cannot catch it. The blank From exercises the fallback to the
// SMTP username.
func (s *HandlerMailerSuite) Test_Send_RecordsFields() {
	env := s.configuredEnv()

	buildconf.SendMailFunc = func(_ string, _ smtp.Auth, _ string, _ []string, _ []byte) error {
		return nil
	}

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/mail",
		map[string]string{
			"appId":   s.app.ID.String(),
			"envId":   env.ID.String(),
			"to":      "someone@acme.com",
			"subject": "Test subject",
			"body":    "Test body",
		},
		s.auth(),
	)

	s.Equal(http.StatusOK, response.Code)

	emails, err := buildconf.MailerStore().Emails(context.Background(), env.ID)
	s.Require().NoError(err)
	s.Require().Len(emails, 1)

	s.Equal("someone@acme.com", emails[0].To)
	s.Equal("noreply@acme.com", emails[0].From, "an omitted from falls back to the SMTP username")
	s.Equal("Test subject", emails[0].Subject)
	s.Equal("Test body", emails[0].Body)
}

func TestHandlerMailerSuite(t *testing.T) {
	suite.Run(t, &HandlerMailerSuite{})
}
