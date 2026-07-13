package buildconf_test

import (
	"net/smtp"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stretchr/testify/suite"
)

type MailerModelSuite struct {
	suite.Suite
}

func (s *MailerModelSuite) AfterTest(_, _ string) {
	buildconf.SendMailFunc = smtp.SendMail
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

func TestMailerModel(t *testing.T) {
	suite.Run(t, &MailerModelSuite{})
}
