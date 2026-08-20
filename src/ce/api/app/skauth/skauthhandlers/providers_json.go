package skauthhandlers

import (
	"context"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
)

const ClientSecretPlaceholder = "****-****-****-****"

// ProvidersJSON renders the environment's sign-in providers together with its
// auth configuration. Client secrets are replaced with ClientSecretPlaceholder
// so the real value never leaves the server.
func ProvidersJSON(ctx context.Context, env *buildconf.Env) (map[string]any, error) {
	providers, err := skauth.NewStore().Providers(ctx, skauth.ProvidersArgs{EnvID: env.ID})

	if err != nil {
		return nil, err
	}

	rendered := map[string]map[string]any{}

	for _, p := range providers {
		rendered[p.Name] = map[string]any{
			"status":   p.Status,
			"clientId": p.Data.ClientID,
		}

		if p.Data.ClientSecret != "" {
			rendered[p.Name]["clientSecret"] = ClientSecretPlaceholder
		}

		if p.Data.FromAddress != "" {
			rendered[p.Name]["fromAddress"] = p.Data.FromAddress
		}
	}

	out := AuthConfigJSON(env.AuthConf)
	out["providers"] = rendered
	out["redirectUrl"] = skauth.RedirectURL()
	out["authUrl"] = skauth.AuthURL()

	return out, nil
}
