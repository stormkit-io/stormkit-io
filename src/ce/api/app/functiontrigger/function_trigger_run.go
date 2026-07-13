package functiontrigger

import (
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type RunParams struct {
	TriggerID types.ID
	Method    string
	URL       string
	Headers   shttp.Headers
	Payload   []byte
}

// Run executes the trigger's HTTP request and returns a log describing the
// request and its response. It does not persist anything.
func Run(p RunParams) (TriggerLog, error) {
	res, err := shttp.NewRequestV2(utils.GetString(p.Method, shttp.MethodGet), p.URL).
		Headers(p.Headers.Make()).
		Payload(p.Payload).
		Do()

	request := map[string]any{
		"url":     p.URL,
		"method":  p.Method,
		"headers": p.Headers,
		"payload": string(p.Payload),
	}

	var response map[string]any

	if res != nil {
		response = map[string]any{
			"code": res.StatusCode,
			"body": res.String(),
		}
	} else if err != nil {
		response = map[string]any{
			"error": err.Error(),
		}
	}

	return TriggerLog{
		TriggerID: p.TriggerID,
		Request:   request,
		Response:  response,
	}, err
}
