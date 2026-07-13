package mailer

import (
	"fmt"
	"net/smtp"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

var SendMailFunc = smtp.SendMail

type SendEmailParams struct {
	To      string
	Subject string
	Body    string
}

// SendEmail sends an HTML email to the given recipient using the system SMTP configuration.
func SendEmail(p SendEmailParams) error {
	cfg := config.Get().SMTP

	if !cfg.IsConfigured() {
		return fmt.Errorf("smtp not configured")
	}

	from := utils.GetString(cfg.From, cfg.Username)
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	msg := []byte(
		fmt.Sprintf("From: %s\n", from) +
			fmt.Sprintf("Subject: %s\n", p.Subject) +
			mime +
			fmt.Sprintf("<html><body>%s</body></html>", p.Body),
	)

	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	port := utils.GetString(cfg.Port, "587")

	return SendMailFunc(cfg.Host+":"+port, auth, from, []string{p.To}, msg)
}
