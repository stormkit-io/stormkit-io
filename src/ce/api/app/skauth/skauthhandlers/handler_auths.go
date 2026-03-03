package skauthhandlers

import (
	"fmt"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

const ClientSecretPlaceholder = "****-****-****-****"

func handlerAuths(req *app.RequestContext) *shttp.Response {
	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("error while fetching environment for skauth: %s, envId: %s", err.Error(), req.EnvID.String()))
	}

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

	successURL := ""
	ttl := 0

	if env.AuthConf != nil {
		successURL = env.AuthConf.SuccessURL
		ttl = env.AuthConf.TTL
	}

	return &shttp.Response{
		Data: map[string]any{
			"providers":   returnValue,
			"successUrl":  successURL,
			"tokenTtl":    ttl,
			"redirectUrl": skauth.RedirectURL(),
			"authUrl":     skauth.AuthURL(),
		},
	}
}
