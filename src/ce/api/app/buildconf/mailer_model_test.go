package buildconf_test

import (
	"context"
	"encoding/json"
	"net/smtp"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stretchr/testify/suite"
)

type MailerModelSuite struct {
	suite.Suite
}

func (s *MailerModelSuite) AfterTest(_, _ string) {
	buildconf.SendMailFunc = buildconf.SendMailWithDeadline
	admin.ResetCache(context.Background())
}

// instanceMailer makes the instance's default mailer the given one.
func (s *MailerModelSuite) instanceMailer(mailer *admin.MailerConfig) {
	admin.SetConfig(&admin.InstanceConfig{MailerConfig: mailer})
}

func (s *MailerModelSuite) defaultMailer() *admin.MailerConfig {
	return &admin.MailerConfig{
		Host:     "smtp.instance.test",
		Port:     "2525",
		Username: "instance-user",
		Password: "instance-secret",
		From:     "Stormkit <noreply@instance.test>",
	}
}

// Test_MarshalJSON_MasksPassword makes masking the default rather than a
// convention: a MailerConf placed in a response cannot leak the credential
// even when the caller never calls JSON.
func (s *MailerModelSuite) Test_MarshalJSON_MasksPassword() {
	mailer := &buildconf.MailerConf{
		Host:     "smtp.gmail.com",
		Port:     "587",
		Username: "test-user",
		Password: "super-secret",
	}

	serialized, err := json.Marshal(map[string]any{"config": mailer})
	s.NoError(err)
	s.NotContains(string(serialized), "super-secret")
	s.Contains(string(serialized), buildconf.PasswordPlaceholder)
}

// Test_Bytes_KeepsRealPassword pins the other half: persistence must keep
// writing the encrypted credential, not the placeholder MarshalJSON renders.
func (s *MailerModelSuite) Test_Bytes_KeepsRealPassword() {
	mailer := &buildconf.MailerConf{
		Host:     "smtp.gmail.com",
		Port:     "587",
		Username: "test-user",
		Password: "super-secret",
	}

	b, err := mailer.Bytes()
	s.NoError(err)
	s.NotContains(string(b), buildconf.PasswordPlaceholder)

	restored := &buildconf.MailerConf{}
	s.NoError(json.Unmarshal(b, restored))
	s.Equal("super-secret", restored.Password)
	s.Equal("test-user", restored.Username)
}

func (s *MailerModelSuite) Test_String_DefaultPort() {
	mailer := &buildconf.MailerConf{
		Host:     "smtp.gmail.com",
		Username: "test",
		Password: "testpwd",
	}

	expected := "smtp://test:testpwd@smtp.gmail.com:587"
	s.Equal(expected, mailer.String())
}

func (s *MailerModelSuite) Test_String_SpecialCharsInCredentials() {
	mailer := &buildconf.MailerConf{
		Host:     "smtp.example.com",
		Port:     "465",
		Username: "user@example.com",
		Password: "p@ss:word/123",
	}

	expected := "smtp://user%40example.com:p%40ss%3Aword%2F123@smtp.example.com:465"
	s.Equal(expected, mailer.String())
}

func (s *MailerModelSuite) Test_Send_EnvelopeExtractsBareAddress() {
	mailer := &buildconf.MailerConf{
		Host:     "smtp.example.com",
		Username: "Triplan <mailer@triplan.to>",
		Password: "secret",
	}

	var capturedFrom string
	var capturedMsg []byte

	buildconf.SendMailFunc = func(_ string, _ smtp.Auth, from string, _ []string, msg []byte) error {
		capturedFrom = from
		capturedMsg = msg
		return nil
	}

	s.NoError(mailer.Send(buildconf.SendEmailParams{
		To:      "user@example.com",
		Subject: "Hello",
		Body:    "Body",
	}))

	s.Equal("mailer@triplan.to", capturedFrom, "envelope must be the bare address")
	s.Contains(string(capturedMsg), "From: Triplan <mailer@triplan.to>", "From header keeps display name")
}

func (s *MailerModelSuite) Test_Send_EnvelopeUsesParamsFromAddress() {
	mailer := &buildconf.MailerConf{
		Host:     "smtp.example.com",
		Username: "smtp-user@example.com",
		Password: "secret",
	}

	var capturedFrom string

	buildconf.SendMailFunc = func(_ string, _ smtp.Auth, from string, _ []string, _ []byte) error {
		capturedFrom = from
		return nil
	}

	s.NoError(mailer.Send(buildconf.SendEmailParams{
		To:      "user@example.com",
		From:    "Acme <noreply@acme.com>",
		Subject: "Hello",
		Body:    "Body",
	}))

	s.Equal("noreply@acme.com", capturedFrom)
}

func (s *MailerModelSuite) Test_Mailer_Custom() {
	custom := &buildconf.MailerConf{Host: "smtp.acme.test", Username: "u", Password: "p"}
	s.instanceMailer(s.defaultMailer())

	s.Same(custom, (&buildconf.Env{MailerConf: custom}).Mailer())
}

func (s *MailerModelSuite) Test_Mailer_UseDefault() {
	s.instanceMailer(s.defaultMailer())

	mailer := (&buildconf.Env{MailerConf: &buildconf.MailerConf{UseDefault: true}}).Mailer()

	s.Require().NotNil(mailer)
	s.Equal("smtp.instance.test", mailer.Host)
	s.Equal("2525", mailer.Port)
	s.Equal("instance-user", mailer.Username)
	s.Equal("instance-secret", mailer.Password)
	s.Equal("Stormkit <noreply@instance.test>", mailer.Sender)
}

// Test_Mailer_UseDefault_NotConfigured verifies that an environment on the
// instance mailer has no mailer while the admin has not configured one, so
// emails are recorded but not sent.
func (s *MailerModelSuite) Test_Mailer_UseDefault_NotConfigured() {
	s.instanceMailer(nil)

	s.Nil((&buildconf.Env{MailerConf: &buildconf.MailerConf{UseDefault: true}}).Mailer())
}

func (s *MailerModelSuite) Test_Mailer_None() {
	s.instanceMailer(s.defaultMailer())

	s.Nil((&buildconf.Env{}).Mailer())
}

// Test_Send_SenderOverridesRequestedFrom verifies that emails sent through the
// instance mailer always come from the instance's sender.
func (s *MailerModelSuite) Test_Send_SenderOverridesRequestedFrom() {
	mailer := &buildconf.MailerConf{
		Host:     "smtp.instance.test",
		Username: "instance-user",
		Password: "instance-secret",
		Sender:   "Stormkit <noreply@instance.test>",
	}

	var capturedFrom string
	var capturedMsg []byte

	buildconf.SendMailFunc = func(_ string, _ smtp.Auth, from string, _ []string, msg []byte) error {
		capturedFrom = from
		capturedMsg = msg
		return nil
	}

	s.NoError(mailer.Send(buildconf.SendEmailParams{
		To:      "user@example.com",
		From:    "Spoof <ceo@instance.test>",
		Subject: "Hello",
		Body:    "Body",
	}))

	s.Equal("noreply@instance.test", capturedFrom)
	s.Contains(string(capturedMsg), "From: Stormkit <noreply@instance.test>")
}

func (s *MailerModelSuite) Test_Bytes_RoundTripsUseDefault() {
	data, err := (&buildconf.MailerConf{UseDefault: true}).Bytes()
	s.Require().NoError(err)

	mailer := &buildconf.MailerConf{}
	s.Require().NoError(json.Unmarshal(data, mailer))
	s.True(mailer.UseDefault)
	s.Equal(true, mailer.JSON()["useDefault"])
}

func TestMailerModel(t *testing.T) {
	suite.Run(t, &MailerModelSuite{})
}
