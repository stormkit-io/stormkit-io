package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/functiontrigger"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerFunctionTriggerLogsGet(req *RequestContext) *shttp.Response {
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

	logs, err := store.Logs(req.Context(), triggerID)

	if err != nil {
		return shttp.Error(err)
	}

	masked := make([]functiontrigger.TriggerLog, len(logs))

	for i, log := range logs {
		masked[i] = log.Masked()
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"logs": masked,
		},
	}
}
