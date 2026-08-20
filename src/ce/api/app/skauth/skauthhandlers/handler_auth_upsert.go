package skauthhandlers

import (
	"errors"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type AuthUpsertRequest struct {
	ProviderName string `json:"providerName"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	FromAddress  string `json:"fromAddress"`
	Status       *bool  `json:"status"`
}

// handlerAuthUpsert handles the upsert of an authentication provider
// configuration for the dashboard.
func handlerAuthUpsert(req *app.RequestContext) *shttp.Response {
	data := AuthUpsertRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	err = UpsertProvider(req.Context(), UpsertProviderParams{
		Env:   env,
		AppID: req.App.ID,
		Data:  data,
	})

	if err != nil {
		var verr *ProviderValidationError

		if errors.As(err, &verr) {
			return shttp.BadRequest(map[string]any{"error": verr.Message})
		}

		return shttp.Error(err)
	}

	return shttp.OK()
}
