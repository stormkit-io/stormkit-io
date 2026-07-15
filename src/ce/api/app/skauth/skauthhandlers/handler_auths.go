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

		if p.Data.FromAddress != "" {
			returnValue[p.Name]["fromAddress"] = p.Data.FromAddress
		}
	}

	successURL := ""
	ttl := 0
	allowedOrigins := []string{}
	oauthServerEnabled := false
	oauthResourcePath := ""
	oauthAllowLoopback := false
	cookieDomain := ""
	loginURL := ""

	if env.AuthConf != nil {
		successURL = env.AuthConf.SuccessURL
		ttl = env.AuthConf.TTL
		oauthServerEnabled = env.AuthConf.OAuthServerEnabled()
		cookieDomain = env.AuthConf.CookieDomain
		loginURL = env.AuthConf.LoginURL

		if env.AuthConf.AllowedOrigins != nil {
			allowedOrigins = env.AuthConf.AllowedOrigins
		}

		if env.AuthConf.OAuthServer != nil {
			oauthResourcePath = env.AuthConf.OAuthServer.ResourcePath
			oauthAllowLoopback = env.AuthConf.OAuthServer.AllowLoopback
		}
	}

	return &shttp.Response{
		Data: map[string]any{
			"providers":          returnValue,
			"successUrl":         successURL,
			"tokenTtl":           ttl,
			"allowedOrigins":     allowedOrigins,
			"oauthServerEnabled": oauthServerEnabled,
			"oauthResourcePath":  oauthResourcePath,
			"oauthAllowLoopback": oauthAllowLoopback,
			"cookieDomain":       cookieDomain,
			"loginUrl":           loginURL,
			"redirectUrl":        skauth.RedirectURL(),
			"authUrl":            skauth.AuthURL(),
		},
	}
}
