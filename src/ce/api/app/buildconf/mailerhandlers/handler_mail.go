package mailerhandlers

import (
	"net/http"
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

func HandlerMail(req *app.RequestContext) *shttp.Response {
	data := RequestData{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	if strings.TrimSpace(data.Body) == "" {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data:   map[string]string{"error": "Email body is a required field."},
		}
	}

	if strings.TrimSpace(data.Subject) == "" {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data:   map[string]string{"error": "Subject is a required field."},
		}
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	config := env.MailerConf
	from := data.From

	if config != nil {
		from = utils.GetString(data.From, config.DefaultFrom())

		if err := config.Send(buildconf.SendEmailParams{
			To:      data.To,
			From:    data.From,
			Subject: data.Subject,
			Body:    data.Body,
		}); err != nil {
			return &shttp.Response{
				Status: http.StatusInternalServerError,
				Data:   map[string]string{"error": err.Error()},
			}
		}
	}

	email := buildconf.Email{
		EnvID:   req.EnvID,
		From:    from,
		To:      data.To,
		Body:    data.Body,
		Subject: data.Subject,
	}

	if err := buildconf.MailerStore().InsertEmail(req.Context(), email); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
