package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/functiontrigger"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerFunctionTriggersGet(req *RequestContext) *shttp.Response {
	triggers, err := functiontrigger.NewStore().List(req.Context(), req.Env.ID)

	if err != nil {
		return shttp.Error(err)
	}

	response := []map[string]any{}

	for _, t := range triggers {
		response = append(response, t.ToMap())
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"triggers": response,
		},
	}
}
