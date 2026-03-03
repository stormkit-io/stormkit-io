package skauthhandlers

import (
	"fmt"
	"net/url"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type AuthConfigUpdateRequest struct {
	SuccessURL string `json:"successUrl"`
	TTL        int    `json:"tokenTtl"`
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

	err := buildconf.NewStore().SaveAuthConf(req.Context(), req.EnvID, &buildconf.SKAuthConf{
		SuccessURL: data.SuccessURL,
		TTL:        data.TTL,
	})

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("error while saving auth conf for env: %s, err=%s", req.EnvID.String(), err.Error()))
	}

	return shttp.OK()
}
