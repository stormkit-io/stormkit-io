package publicapiv1

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/functiontrigger"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerFunctionTriggerDelete(req *RequestContext) *shttp.Response {
	triggerID := utils.StringToID(req.Query().Get("triggerId"))

	if triggerID == 0 {
		return shttp.NotFound()
	}

	store := functiontrigger.NewStore()

	trigger, err := store.ByID(req.Context(), triggerID)

	if err != nil {
		return shttp.Error(err)
	}

	if trigger == nil || trigger.EnvID != req.Env.ID {
		return shttp.NotFound()
	}

	if err := store.Delete(req.Context(), triggerID); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
