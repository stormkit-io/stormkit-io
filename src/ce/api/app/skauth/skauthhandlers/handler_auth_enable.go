package skauthhandlers

import (
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type AuthEnableRequest struct {
	ProviderName string `json:"providerName"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Status       bool   `json:"status"`
}

func handlerAuthEnable(req *app.RequestContext) *shttp.Response {
	data := &AuthEnableRequest{}

	if err := req.Post(data); err != nil {
		return shttp.Error(err)
	}

	ctx := req.Context()
	env, err := buildconf.NewStore().EnvironmentByID(ctx, req.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	if env.SchemaConf == nil {
		return shttp.BadRequest(map[string]any{
			"error": "Schema configuration is not set for this environment. Please configure it first.",
		})
	}

	data.ProviderName = strings.TrimSpace(strings.ToLower(data.ProviderName))

	provider := GetProviderClient(data.ProviderName, data.ClientID, data.ClientSecret)

	if provider == nil {
		return shttp.BadRequest(map[string]any{
			"error": "Invalid provider",
		})
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

	err = skauth.NewStore().SaveProvider(ctx, skauth.SaveProviderArgs{
		EnvID:  req.EnvID,
		AppID:  req.App.ID,
		Status: data.Status,
		Client: provider,
	})

	if err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
