package adminhandlers

import (
	"net/mail"
	"strconv"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// AdminMailerSetRequest configures the instance's default mailer.
type AdminMailerSetRequest struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"` // empty or the placeholder keeps the stored one
	From     string `json:"from"`     // e.g. Acme <noreply@acme.com>
}

// handlerAdminMailerSet creates or replaces the instance's default mailer.
// Environments that opted into it start using the new settings right away.
//
// The stored password is kept when none is sent, but only while the host and
// username stay the same, so the credential cannot be sent to another server.
func handlerAdminMailerSet(req *user.RequestContext) *shttp.Response {
	data := AdminMailerSetRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	cnf := admin.MustConfig().Clone()
	mailer := admin.MailerConfig{
		Host:     strings.TrimSpace(data.Host),
		Port:     strings.TrimSpace(data.Port),
		Username: strings.TrimSpace(data.Username),
		Password: data.Password,
		From:     strings.TrimSpace(data.From),
	}

	if mailer.Password == "" || mailer.Password == admin.MailerPasswordPlaceholder {
		mailer.Password = ""

		if prev := cnf.MailerConfig; prev != nil && prev.Host == mailer.Host && prev.Username == mailer.Username {
			mailer.Password = prev.Password
		}
	}

	errors := map[string]string{}

	if mailer.Host == "" {
		errors["host"] = "SMTP Host is a required field."
	}

	if mailer.Username == "" {
		errors["username"] = "Username is a required field."
	}

	if mailer.Password == "" {
		errors["password"] = "Password is a required field."
	}

	if _, err := mail.ParseAddress(mailer.From); err != nil {
		errors["from"] = "From must be an email address, optionally with a name: Acme <noreply@acme.com>."
	}

	if mailer.Port != "" {
		if port, err := strconv.Atoi(mailer.Port); err != nil || port < 1 || port > 65535 {
			errors["port"] = "SMTP Port must be a number between 1 and 65535."
		}
	}

	if len(errors) > 0 {
		return shttp.BadRequest(map[string]any{"errors": errors})
	}

	cnf.MailerConfig = &mailer

	if err := admin.Store().UpsertConfig(req.Context(), cnf); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Data: map[string]any{
			"mailer": mailer.JSON(),
		},
	}
}
