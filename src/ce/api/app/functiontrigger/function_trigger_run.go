package functiontrigger

import (
	"io"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// maxLoggedBodySize bounds how much of a target's response is kept in the log
// row. The target is an arbitrary user-supplied host and an unavailable one
// produces several rows per tick, so an uncapped body is a storage amplifier.
const maxLoggedBodySize = 64 * 1024

// loggedBody reads at most maxLoggedBodySize of the response body, marking the
// value when the target sent more than that.
func loggedBody(res *shttp.HTTPResponse) string {
	body, _ := io.ReadAll(io.LimitReader(res.Body, maxLoggedBodySize+1))

	if len(body) > maxLoggedBodySize {
		return string(body[:maxLoggedBodySize]) + "... (truncated)"
	}

	return string(body)
}

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
			"body": loggedBody(res),
		}
	} else if err != nil {
		response = map[string]any{
			"error": err.Error(),
		}
	}

	// The row's created_at is filled in by the database on insert, but callers
	// that return the log straight to a client need it populated here too.
	return TriggerLog{
		TriggerID: p.TriggerID,
		Request:   request,
		Response:  response,
		CreatedAt: utils.NewUnix(),
	}, err
}
