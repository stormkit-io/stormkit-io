package mailer_test

import (
	"net/smtp"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/mailer"
	"github.com/stretchr/testify/suite"
)

type MailerSuite struct {
	suite.Suite
}

func (s *MailerSuite) AfterTest(_, _ string) {
	mailer.SendMailFunc = smtp.SendMail
}

func (s *MailerSuite) Test_SendEmail_Success() {
	called := false

	config.SetSMTP(&config.SMTPConfig{
		Host:     "smtp.example.com",
		Port:     "587",
		Username: "user@example.com",
		Password: "secret",
		From:     "Stormkit <no-reply@stormkit.io>",
	})

	mailer.SendMailFunc = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
		s.Equal("smtp.example.com:587", addr)
		s.Equal("user@example.com", from)
		s.Equal([]string{"customer@example.com"}, to)
		s.Equal([]byte(
			"From: Stormkit <no-reply@stormkit.io>\n"+
				"Subject: Your license key\n"+
				mime+
				"<html><body>Here is your key.</body></html>",
		), msg)
		called = true
		return nil
	}

	err := mailer.SendEmail(mailer.SendEmailParams{
		To:      "customer@example.com",
		Subject: "Your license key",
		Body:    "Here is your key.",
	})

	s.NoError(err)
	s.True(called)
}

func (s *MailerSuite) Test_SendEmail_DefaultPort() {
	config.SetSMTP(&config.SMTPConfig{
		Host:     "smtp.example.com",
		Username: "user@example.com",
		Password: "secret",
	})

	mailer.SendMailFunc = func(addr string, _ smtp.Auth, _ string, _ []string, _ []byte) error {
		s.Equal("smtp.example.com:587", addr)
		return nil
	}

	s.NoError(mailer.SendEmail(mailer.SendEmailParams{
		To:      "customer@example.com",
		Subject: "Test",
		Body:    "Body",
	}))
}

func (s *MailerSuite) Test_SendEmail_NotConfigured() {
	config.SetSMTP(&config.SMTPConfig{})

	err := mailer.SendEmail(mailer.SendEmailParams{
		To:      "customer@example.com",
		Subject: "Test",
		Body:    "Body",
	})

	s.EqualError(err, "smtp not configured")
}

func TestMailerSuite(t *testing.T) {
	suite.Run(t, new(MailerSuite))
}
