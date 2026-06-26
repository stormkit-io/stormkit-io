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

	if errs := validateFunctionTrigger(tf); errs != nil {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data:   errs,
		}
	}

	store := functiontrigger.NewStore()
	existing, err := store.ByID(req.Context(), tf.ID)

	if err != nil {
		return shttp.Error(err)
	}

	if existing == nil || existing.EnvID != req.Env.ID {
		return shttp.NotFound()
	}

	if err := store.Update(req.Context(), tf.toRecord(req.Env.ID)); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
