package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/functiontrigger"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

func handlerFunctionTriggerInvoke(req *RequestContext) *shttp.Response {
	body := struct {
		ID types.ID `json:"id,string"`
	}{}

	if err := req.Post(&body); err != nil {
		return shttp.Error(err)
	}

	if body.ID == 0 {
		return shttp.NotFound()
	}

	store := functiontrigger.NewStore()

	trigger, err := store.ByID(req.Context(), body.ID)

	if err != nil {
		return shttp.Error(err)
	}

	if trigger == nil || trigger.EnvID != req.Env.ID {
		return shttp.NotFound()
	}

	log, _ := functiontrigger.Run(functiontrigger.RunParams{
		TriggerID: trigger.ID,
		Method:    trigger.Options.Method,
		URL:       trigger.Options.URL,
		Headers:   trigger.Options.Headers,
		Payload:   trigger.Options.Payload,
	})

	if err := store.InsertLogs(req.Context(), []functiontrigger.TriggerLog{log}); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"log": log.Masked(),
		},
	}
}
