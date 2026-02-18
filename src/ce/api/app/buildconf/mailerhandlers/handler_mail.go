package mailerhandlers

import (
	"fmt"
	"net/http"
	"net/smtp"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type RequestData struct {
	To      string `json:"to"`
	From    string `json:"from"`
	Body    string `json:"body"`
	Subject string `json:"subject"`
}

var SendMail = smtp.SendMail

func HandlerMail(req *app.RequestContext) *shttp.Response {
	data := RequestData{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	if strings.TrimSpace(data.Body) == "" {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "Email body is a required field.",
			},
		}
	}

	if strings.TrimSpace(data.Subject) == "" {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "Subject is a required field.",
			},
		}
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	config := env.MailerConf

	if config == nil {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "Mailer is not yet configured.",
			},
		}
	}

	// Set the sender, recipient, and message.
	to := strings.Split(data.To, ";")

	for i := range to {
		to[i] = strings.TrimSpace(to[i])
	}

	fromHeader := fmt.Sprintf("From: %s\n", utils.GetString(data.From, config.Username))
	subject := fmt.Sprintf("Subject: %s\n", data.Subject)
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body := fmt.Sprintf("<html><body>%s</body></html>", data.Body)
	msg := []byte(fromHeader + subject + mime + body)

	// Combine the host and port
	addr := config.Host + ":" + utils.GetString(config.Port, "587")
	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)

	// Send the email
	if err = SendMail(addr, auth, config.Username, to, msg); err != nil {
		return &shttp.Response{
			Status: http.StatusInternalServerError,
			Data: map[string]string{
				"error": err.Error(),
			},
		}
	}

	// Store the sent email in the database
	email := buildconf.Email{
		EnvID:   req.EnvID,
		From:    utils.GetString(data.From, config.Username),
		To:      data.To,
		Body:    data.Body,
		Subject: data.Subject,
	}

	if err := buildconf.MailerStore().InsertEmail(req.Context(), email); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
