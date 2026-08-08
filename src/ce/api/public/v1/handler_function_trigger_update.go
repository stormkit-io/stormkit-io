package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/functiontrigger"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerFunctionTriggerUpdate(req *RequestContext) *shttp.Response {
	tf := &functionTriggerRequest{}

	if err := req.Post(tf); err != nil {
		return shttp.Error(err)
	}

	store := functiontrigger.NewStore()
	existing, err := store.ByID(req.Context(), tf.ID)

	if err != nil {
		return shttp.Error(err)
	}

	if existing == nil || existing.EnvID != req.Env.ID {
		return shttp.NotFound()
	}

	// A partial update: the fields the body carries are applied to the stored
	// trigger and the rest keep their current values, so a caller that only
	// means to change the cron cannot destroy headers or documentation it never
	// mentioned.
	tf.applyTo(existing)

	if errs := validateFunctionTrigger(existing); errs != nil {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data:   errs,
		}
	}

	if err := store.Update(req.Context(), existing); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
