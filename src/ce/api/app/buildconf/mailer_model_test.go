package buildconf_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stretchr/testify/suite"
)

type MailerModelSuite struct {
	suite.Suite
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

func TestMailerModel(t *testing.T) {
	suite.Run(t, &MailerModelSuite{})
}
