package publicapiv1

import (
	"errors"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/mailerhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerMailSend sends an email through the environment's SMTP configuration
// and appends it to the mailer log.
func handlerMailSend(req *RequestContext) *shttp.Response {
	data := mailerhandlers.RequestData{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	delivered, err := mailerhandlers.SendAndRecord(req.Context(), mailerhandlers.SendAndRecordParams{
		Env:  req.Env,
		Data: data,
	})

	if err != nil {
		var verr *mailerhandlers.SendValidationError

		if errors.As(err, &verr) {
			return shttp.BadRequest(map[string]any{"error": verr.Message})
		}

		// shttp.Error logs with caller info and returns a generic body. The
		// SMTP host is caller-supplied, so echoing the raw error would turn a
		// failed send into a probe oracle for hosts reachable from the API.
		return shttp.Error(err)
	}

	// delivered is false when the environment has no SMTP configuration: the
	// message is recorded but never sent, so reporting a bare ok would be a
	// false positive on the one call used to verify a mailer works.
	return &shttp.Response{
		Status: http.StatusOK,
		Data:   map[string]any{"ok": true, "delivered": delivered},
	}
}
