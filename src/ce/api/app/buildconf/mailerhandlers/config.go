package mailerhandlers

import (
	"strconv"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"
)

// ConfigUpdateRequest patches the mailer configuration of an environment. Nil
// fields keep their stored value, which lets the public API and the MCP tool
// share one payload shape.
type ConfigUpdateRequest struct {
	Username *string `json:"username"`
	Password *string `json:"password"`
	SMTPHost *string `json:"smtpHost"`
	SMTPPort *string `json:"smtpPort"`
}

// ApplyConfigUpdate validates data and merges its provided fields into conf in
// place. It performs no I/O so it can be shared by the public API handler and
// the MCP tool. On validation failure it returns a
// *shttperr.ValidationError, which shttp.Error renders as a 400.
//
// An omitted, empty or placeholder password keeps the stored one: the password
// is write-only across every surface, so clients that never see it can still
// submit the rest of the form.
//
// Retaining it is only safe while the credential keeps pointing at the same
// account, so changing the host or the username drops the stored password.
// Otherwise a caller who is never allowed to read it could repoint the config
// at a server it controls and have Send hand the credential over.
func ApplyConfigUpdate(conf *buildconf.MailerConf, data ConfigUpdateRequest) error {
	previousHost := conf.Host
	previousUsername := conf.Username

	if data.SMTPHost != nil {
		conf.Host = strings.TrimSpace(*data.SMTPHost)
	}

	if data.Username != nil {
		conf.Username = strings.TrimSpace(*data.Username)
	}

	if conf.Host != previousHost || conf.Username != previousUsername {
		conf.Password = ""
	}

	if data.Password != nil && *data.Password != "" && *data.Password != buildconf.PasswordPlaceholder {
		conf.Password = *data.Password
	}

	if data.SMTPPort != nil {
		conf.Port = strings.TrimSpace(*data.SMTPPort)
	}

	errors := map[string]string{}

	if conf.Host == "" {
		errors["host"] = "SMTP Host is a required field."
	}

	if conf.Username == "" {
		errors["username"] = "Username is a required field."
	}

	if conf.Password == "" {
		errors["password"] = "Password is a required field."
	}

	if conf.Port != "" {
		port, err := strconv.Atoi(conf.Port)

		if err != nil || port < 1 || port > 65535 {
			errors["port"] = "SMTP Port must be a number between 1 and 65535."
		}
	}

	if len(errors) > 0 {
		return &shttperr.ValidationError{Errors: errors}
	}

	return nil
}
