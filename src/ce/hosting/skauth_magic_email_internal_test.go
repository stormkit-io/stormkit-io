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

	link := `https://acme.com/_stormkit/auth/magic?token=a&amp;b=&#34;&lt;c&gt;`

	s.Equal("Your magic link", msg.subject())
	s.Contains(msg.body(), "Sign in to acme.com</h1>")
	s.Contains(msg.body(), `href="`+link+`"`)
	s.Contains(msg.body(), ">Sign in</a>")
	s.Contains(msg.body(), ">"+link+"</a>")
	s.Contains(msg.body(), "expires in 15 minutes")
	s.NotContains(msg.body(), "%!")
}

// Test_SiteName verifies the email names the site the user lands on: a web
// redirect origin when there is one, the link's host otherwise.
func (s *MagicLinkEmailSuite) Test_SiteName() {
	s.Equal("app.acme.com", siteName("https://app.acme.com", "api.acme.com"))
	s.Equal("localhost:3000", siteName("http://localhost:3000", "api.acme.com"))
	s.Equal("api.acme.com", siteName("", "api.acme.com"))
	s.Equal("api.acme.com", siteName("myapp://auth", "api.acme.com"))
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
