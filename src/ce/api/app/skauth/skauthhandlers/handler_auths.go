package skauthhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

const ClientSecretPlaceholder = "****-****-****-****"

func handlerAuths(req *app.RequestContext) *shttp.Response {
	providers, err := skauth.NewStore().Providers(req.Context(), skauth.ProvidersArgs{
		EnvID: req.EnvID,
	})

	if err != nil {
		return shttp.Error(err)
	}

	returnValue := map[string]map[string]any{}

	for _, p := range providers {
		returnValue[p.Name] = map[string]any{
			"status":   p.Status,
			"clientId": p.Data.ClientID,
		}

		if p.Data.ClientSecret != "" {
			returnValue[p.Name]["clientSecret"] = ClientSecretPlaceholder
		}
	}

	return &shttp.Response{
		Data: map[string]any{
			"providers":   returnValue,
			"redirectUrl": skauth.RedirectURL(),
		},
	}
}
