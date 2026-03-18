package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerEnvPull(req *RequestContext) *shttp.Response {
	return &shttp.Response{
		Status: http.StatusOK,
		Data:   req.Env.Data.Vars,
	}
}
