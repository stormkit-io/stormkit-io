package mailerhandlers_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/mailerhandlers"
	"github.com/stretchr/testify/suite"
)

type EmailsJSONSuite struct {
	suite.Suite
}

func (s *EmailsJSONSuite) Test_OmitsBodyAndMasksRecipient() {
	rendered := mailerhandlers.EmailsJSON([]*buildconf.Email{{
		To:      "jane@example.com",
		From:    "noreply@acme.com",
		Subject: "Your magic link",
		Body:    `<a href="https://acme.com/_stormkit/auth/magic?token=secret">link</a>`,
	}})

	s.Require().Len(rendered, 1)
	s.NotContains(rendered[0], "body")
	s.Equal("j***@example.com", rendered[0]["to"])
	s.Equal("Your magic link", rendered[0]["subject"])

	// The sender is the app's own address, not end-user data.
	s.Equal("noreply@acme.com", rendered[0]["from"])
}

func (s *EmailsJSONSuite) Test_MaskRecipients() {
	s.Equal("j***@example.com", mailerhandlers.MaskRecipients("jane@example.com"))
	s.Equal("a***@acme.com; b***@acme.com", mailerhandlers.MaskRecipients("ann@acme.com;bob@acme.com"))
	s.Equal("", mailerhandlers.MaskRecipients(""))

	// The first rune survives, not the first byte: a byte slice would split a
	// multi-byte local part and serialize as U+FFFD.
	s.Equal("\u00f1***@example.com", mailerhandlers.MaskRecipients("\u00f1o\u00f1o@example.com"))

	// A malformed address is masked whole rather than passed through.
	s.Equal("***", mailerhandlers.MaskRecipients("not-an-address"))
	s.Equal("***", mailerhandlers.MaskRecipients("@example.com"))
}

func TestEmailsJSONSuite(t *testing.T) {
	suite.Run(t, &EmailsJSONSuite{})
}
