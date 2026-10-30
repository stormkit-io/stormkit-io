package authwall

import (
	"errors"
	"fmt"
	"html"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/mailer"
)

// MagicLinkEmail sends sign-in links with the instance's own email: the
// system SMTP server when one is configured, the instance's default mailer
// otherwise.
type MagicLinkEmail struct{}

// Available reports whether sign-in links can be sent. The login form only
// offers them when they can.
func (MagicLinkEmail) Available() bool {
	return config.Get().SMTP.IsConfigured() || buildconf.InstanceMailer() != nil
}

// SendMagicLinkParams are the parameters of Send.
type SendMagicLinkParams struct {
	To string

	// Name names the protected environment, such as "storefront / production".
	Name string

	// URL is the sign-in link.
	URL string
}

// Send emails the sign-in link.
func (m MagicLinkEmail) Send(p SendMagicLinkParams) error {
	subject := fmt.Sprintf("Sign in to %s", p.Name)
	body := fmt.Sprintf(
		`<p>Use the link below to sign in to <strong>%s</strong>. It works once, within %d minutes.</p>`+
			`<p><a href="%s">Sign in to %s</a></p>`+
			`<p>If you didn't ask for this link, you can ignore this email.</p>`,
		html.EscapeString(p.Name), MagicLinkMaxMins, html.EscapeString(p.URL), html.EscapeString(p.Name),
	)

	if config.Get().SMTP.IsConfigured() {
		return mailer.SendEmail(mailer.SendEmailParams{To: p.To, Subject: subject, Body: body})
	}

	if mc := buildconf.InstanceMailer(); mc != nil {
		return mc.Send(buildconf.SendEmailParams{To: p.To, Subject: subject, Body: body})
	}

	return errors.New("no mailer is configured")
}
