package adminhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// AdminMailerTestRequest names the recipient of a test email.
type AdminMailerTestRequest struct {
	To string `json:"to"`
}

// handlerAdminMailerTestSend sends a test email through the instance's default
// mailer. Unlike the environment mailer, it returns the SMTP error: only the
// instance admin can call it, and they need it to fix the settings.
func handlerAdminMailerTestSend(req *user.RequestContext) *shttp.Response {
	data := AdminMailerTestRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	if !utils.IsValidEmail(data.To) {
		return shttp.BadRequest(map[string]any{"error": "Recipient must be an email address."})
	}

	mailer := buildconf.InstanceMailer()

	if mailer == nil {
		return shttp.BadRequest(map[string]any{"error": "Configure the default mailer first."})
	}

	err := mailer.Send(buildconf.SendEmailParams{
		To:      data.To,
		Subject: "Stormkit test email",
		Body:    "<p>The default mailer of your Stormkit instance works.</p>",
	})

	if err != nil {
		return &shttp.Response{
			Status: http.StatusBadGateway,
			Data:   map[string]any{"error": "The SMTP server refused the email: " + err.Error()},
		}
	}

	return shttp.OK()
}
