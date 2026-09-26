package hosting

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stretchr/testify/suite"
)

type MagicLinkEmailSuite struct {
	suite.Suite
}

func (s *MagicLinkEmailSuite) email(data skauth.ProviderData) magicLinkEmail {
	return magicLinkEmail{
		Provider: &skauth.Provider{Name: skauth.ProviderMagicLink, Data: data},
		Host:     "acme.com",
		Link:     `https://acme.com/_stormkit/auth/magic?token=a&b="<c>`,
	}
}

func (s *MagicLinkEmailSuite) Test_Default() {
	msg := s.email(skauth.ProviderData{})

	s.Equal("Your magic link", msg.subject())
	s.Contains(msg.body(), "sign in to acme.com")
	s.Contains(msg.body(), `href="https://acme.com/_stormkit/auth/magic?token=a&amp;b=&#34;&lt;c&gt;"`)
	s.Contains(msg.body(), ">Sign in</a>")
	s.Contains(msg.body(), "expires in 15 minutes")
}

func (s *MagicLinkEmailSuite) Test_CustomSubject() {
	s.Equal("Sign in to Acme", s.email(skauth.ProviderData{Subject: "Sign in to Acme"}).subject())
}

// Test_SubjectHeaderInjection verifies a stored subject cannot smuggle extra
// SMTP headers through newlines.
func (s *MagicLinkEmailSuite) Test_SubjectHeaderInjection() {
	msg := s.email(skauth.ProviderData{Subject: "Hi\r\nBcc: attacker@evil.com"})

	s.Equal("HiBcc: attacker@evil.com", msg.subject())
}

// Test_CustomBody_EscapesLink verifies every placeholder is replaced with the
// HTML-escaped link while the owner's template is left as written.
func (s *MagicLinkEmailSuite) Test_CustomBody_EscapesLink() {
	msg := s.email(skauth.ProviderData{Body: `<b>Hi</b> <a href="{{link}}">Go</a> {{link}}`})

	link := `https://acme.com/_stormkit/auth/magic?token=a&amp;b=&#34;&lt;c&gt;`

	s.Equal(`<b>Hi</b> <a href="`+link+`">Go</a> `+link, msg.body())
}

func TestMagicLinkEmailSuite(t *testing.T) {
	suite.Run(t, new(MagicLinkEmailSuite))
}
