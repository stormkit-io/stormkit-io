package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerAppGet returns a single application by its ID.
// Authentication requires a an API key with at least app scope (user-scoped or team-scoped).
func handlerAppGet(req *RequestContext) *shttp.Response {
	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"app": req.App.JSON(),
		},
	}
}
