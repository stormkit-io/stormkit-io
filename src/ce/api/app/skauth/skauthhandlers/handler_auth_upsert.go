package skauthhandlers

import (
	"fmt"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type AuthUpsertRequest struct {
	ProviderName string `json:"providerName"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Status       bool   `json:"status"`
}

// handlerAuthUpsert handles the upsert of an authentication provider configuration.
func handlerAuthUpsert(req *app.RequestContext) *shttp.Response {
	data := &AuthUpsertRequest{}

	if err := req.Post(data); err != nil {
		return shttp.Error(err)
	}

	ctx := req.Context()
	envStore := buildconf.NewStore()
	env, err := envStore.EnvironmentByID(ctx, req.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	if env.SchemaConf == nil {
		return shttp.BadRequest(map[string]any{
			"error": "Schema configuration is not set for this environment. Please configure it first.",
		})
	}

	// Let's create an auth conf on the fly, with default settings
	if env.AuthConf == nil {
		env.AuthConf = &buildconf.SKAuthConf{
			TTL:        10,
			SuccessURL: "/",
			Secret:     utils.RandomToken(128),
			Status:     true,
		}

		if err := envStore.SaveAuthConf(req.Context(), req.EnvID, env.AuthConf); err != nil {
			return shttp.Error(err, fmt.Sprintf("error while saving auth config: %s, envId: %s", err.Error(), req.EnvID.String()))
		}

		if err := appcache.Service().Reset(req.EnvID); err != nil {
			return shttp.Error(err)
		}
	}

	data.ProviderName = strings.TrimSpace(strings.ToLower(data.ProviderName))

	// If updating an existing provider and client secret is not provided, retain the existing secret
	if data.ClientSecret == "" || data.ClientSecret == ClientSecretPlaceholder {
		existingProvider, err := skauth.NewStore().Provider(ctx, req.EnvID, data.ProviderName)

		if err != nil {
			return shttp.Error(err)
		}

		if existingProvider != nil && existingProvider.Data.ClientSecret != "" {
			data.ClientSecret = existingProvider.Data.ClientSecret
		}
	}

	if data.ClientID == "" {
		return shttp.BadRequest(map[string]any{
			"error": "Client ID is required",
		})
	}

	if data.ClientSecret == "" {
		return shttp.BadRequest(map[string]any{
			"error": "Client Secret is required",
		})
	}

	migrationStore, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeMigrations)

	if err != nil {
		return shttp.Error(err)
	}

	defer migrationStore.Close()

	// This is idempotent - if the table already exists, no error is returned.
	if err := migrationStore.CreateAuthTable(ctx); err != nil {
		return shttp.Error(err)
	}

	provider := &skauth.Provider{
		Name:   data.ProviderName,
		Status: data.Status,
		Data: skauth.ProviderData{
			ClientID:     data.ClientID,
			ClientSecret: data.ClientSecret,
		},
	}

	if provider.Client() == nil {
		return shttp.BadRequest(map[string]any{
			"error": "Invalid provider",
		})
	}

	err = skauth.NewStore().SaveProvider(ctx, skauth.SaveProviderArgs{
		EnvID:    req.EnvID,
		AppID:    req.App.ID,
		Provider: provider,
	})

	if err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
