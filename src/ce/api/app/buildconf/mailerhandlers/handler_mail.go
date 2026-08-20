package mailerhandlers

import (
	"errors"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func HandlerMail(req *app.RequestContext) *shttp.Response {
	data := RequestData{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	// WithApp only asserts that an envId was provided, not that it exists, and
	// the store returns (nil, nil) for an unknown id. SendAndRecord
	// dereferences Env, so this has to be checked before the call.
	if env == nil {
		return shttp.NotFound()
	}

	delivered, err := SendAndRecord(req.Context(), SendAndRecordParams{Env: env, Data: data})

	if err != nil {
		var verr *SendValidationError

		if errors.As(err, &verr) {
			return &shttp.Response{
				Status: http.StatusBadRequest,
				Data:   map[string]string{"error": verr.Message},
			}
		}

		// shttp.Error logs with caller info and returns a generic body. The
		// SMTP host is configurable, so the raw error would echo whatever the
		// relay - or the database - said back to the client.
		return shttp.Error(err)
	}

	// delivered is false when the environment has no SMTP configuration: the
	// message is recorded but never sent, so a bare ok would let the dashboard
	// report a successful test email that was never handed to a relay.
	return &shttp.Response{
		Status: http.StatusOK,
		Data:   map[string]any{"ok": true, "delivered": delivered},
	}
}
