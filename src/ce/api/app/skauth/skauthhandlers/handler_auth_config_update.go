package skauthhandlers

import (
	"fmt"
	"net/url"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type AuthConfigUpdateRequest struct {
	SuccessURL string `json:"successUrl"`
	TTL        int    `json:"tokenTtl"`
	Status     bool   `json:"status"`
}

func handlerAuthConfigUpdate(req *app.RequestContext) *shttp.Response {
	data := AuthConfigUpdateRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err, fmt.Sprintf("error while unmarshaling auth config request: %s", err.Error()))
	}

	if data.SuccessURL != "" {
		parsed, err := url.Parse(data.SuccessURL)

		if err != nil {
			return shttp.BadRequest(map[string]any{
				"error": "Success URL format is not valid. Make sure to provide a relative URL.",
				"hint":  "Provide a relative URL such as: /success",
			})
		}

		if parsed.IsAbs() {
			return shttp.BadRequest(map[string]any{
				"error": "Success URL is not a relative URL.",
				"hint":  "Provide a relative URL such as: /success",
			})
		}
	}

	store := buildconf.NewStore()
	env, err := store.EnvironmentByID(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("error while fetching environment for auth config update: %s, envId: %s", err.Error(), req.EnvID.String()))
	}

	if env.AuthConf == nil {
		env.AuthConf = &buildconf.SKAuthConf{
			Secret: utils.RandomToken(128),
		}
	}

	env.AuthConf.SuccessURL = data.SuccessURL
	env.AuthConf.TTL = data.TTL
	env.AuthConf.Status = data.Status

	err = buildconf.NewStore().SaveAuthConf(req.Context(), req.EnvID, env.AuthConf)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("error while saving auth conf for env: %s, err=%s", req.EnvID.String(), err.Error()))
	}

	return shttp.OK()
}
