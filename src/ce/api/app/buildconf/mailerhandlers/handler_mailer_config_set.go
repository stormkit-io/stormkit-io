package mailerhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type ConfigRequestData struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	SMTPHost    string `json:"smtpHost"`
	SMTPPort    string `json:"smtpPort"`
	FromAddress string `json:"fromAddress"`
}

func validateConfig(data ConfigRequestData) map[string]string {
	errors := map[string]string{}

	if data.Username == "" {
		errors["username"] = "Username is a required field."
	}

	if data.Password == "" {
		errors["password"] = "Password is a required field."
	}

	if data.SMTPHost == "" {
		errors["host"] = "SMTP Host is a required field."
	}

	if len(errors) == 0 {
		return nil
	}

	return errors
}

func HandlerMailerConfigSet(req *app.RequestContext) *shttp.Response {
	data := ConfigRequestData{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	if err := validateConfig(data); err != nil {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]any{
				"errors": err,
			},
		}
	}

	config := &buildconf.MailerConf{
		Host:     data.SMTPHost,
		Port:     data.SMTPPort,
		Username: data.Username,
		Password: data.Password,
		EnvID:    req.EnvID,
	}

	if err := buildconf.MailerStore().UpsertConfig(req.Context(), config); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"config": config,
		},
	}
}
