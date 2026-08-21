package publicapiv1

import (
	"errors"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth/skauthhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// authConfigEnabled mirrors the availability of Stormkit Auth itself: the
// feature ships on dev and self-hosted builds only, so its configuration API
// and MCP tools are gated the same way.
func authConfigEnabled() bool {
	return config.IsDevelopment() || config.IsSelfHosted()
}

// handlerAuthConfigGet returns the Stormkit Auth configuration for the
// environment. Secrets — the signing secret and provider client secrets — are
// never included.
func handlerAuthConfigGet(req *RequestContext) *shttp.Response {
	return &shttp.Response{
		Status: http.StatusOK,
		Data:   skauthhandlers.AuthConfigJSON(req.Env.AuthConf),
	}
}

// handlerAuthConfigSet patches the Stormkit Auth configuration for the
// environment. Only the fields present in the request body are changed; the
// rest keep their stored values.
func handlerAuthConfigSet(req *RequestContext) *shttp.Response {
	data := skauthhandlers.AuthConfigUpdateRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	env := req.Env

	if env.AuthConf == nil {
		env.AuthConf = &buildconf.SKAuthConf{
			Secret: utils.RandomToken(128),
		}
	}

	if err := skauthhandlers.ApplyConfigUpdate(env.AuthConf, data); err != nil {
		var verr *skauthhandlers.ConfigValidationError

		if errors.As(err, &verr) {
			return shttp.BadRequest(map[string]any{"error": verr.Message, "hint": verr.Hint})
		}

		return shttp.Error(err)
	}

	if err := buildconf.NewStore().SaveAuthConf(req.Context(), env.ID, env.AuthConf); err != nil {
		return shttp.Error(err)
	}

	// Invalidate the hosting cache so the SKAuth middleware picks up the new
	// settings on the next request instead of serving the stale cached config.
	if err := appcache.Service().Reset(env.ID); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   skauthhandlers.AuthConfigJSON(env.AuthConf),
	}
}
