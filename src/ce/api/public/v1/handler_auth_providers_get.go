package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth/skauthhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerAuthProvidersGet returns the environment's sign-in providers along
// with its auth configuration. Client secrets are never included.
func handlerAuthProvidersGet(req *RequestContext) *shttp.Response {
	data, err := skauthhandlers.ProvidersJSON(req.Context(), req.Env)

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{Status: http.StatusOK, Data: data}
}
