package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerRedirectsGet(req *RequestContext) *shttp.Response {
	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"redirects": req.Env.Data.Redirects,
		},
	}
}
