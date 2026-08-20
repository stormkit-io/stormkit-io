package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/mailerhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerMailerEmailsGet returns the last 100 emails recorded for the
// environment. Message bodies are never included — see EmailsJSON.
func handlerMailerEmailsGet(req *RequestContext) *shttp.Response {
	emails, err := buildconf.MailerStore().Emails(req.Context(), req.Env.ID)

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   map[string]any{"emails": mailerhandlers.EmailsJSON(emails)},
	}
}
