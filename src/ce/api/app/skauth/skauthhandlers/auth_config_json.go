package skauthhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
)

// AuthConfigJSON renders the non-secret Stormkit Auth configuration fields.
// The signing secret and provider client secrets are never included.
func AuthConfigJSON(conf *buildconf.SKAuthConf) map[string]any {
	out := map[string]any{
		"status":             false,
		"successUrl":         "",
		"tokenTtl":           0,
		"allowedOrigins":     []string{},
		"oauthServerEnabled": false,
		"oauthResourcePath":  "",
		"oauthAllowLoopback": false,
		"cookieDomain":       "",
		"loginUrl":           "",
	}

	if conf == nil {
		return out
	}

	out["status"] = conf.Status
	out["successUrl"] = conf.SuccessURL
	out["tokenTtl"] = conf.TTL
	out["cookieDomain"] = conf.CookieDomain
	out["loginUrl"] = conf.LoginURL

	if conf.AllowedOrigins != nil {
		out["allowedOrigins"] = conf.AllowedOrigins
	}

	// Report the effective value the runtime honours, not the stored flag:
	// the OAuth server is inert while Stormkit Auth itself is off, so a raw
	// flag would show the toggle on for an endpoint that rejects every request.
	out["oauthServerEnabled"] = conf.OAuthServerEnabled()

	if conf.OAuthServer != nil {
		out["oauthResourcePath"] = conf.OAuthServer.ResourcePath
		out["oauthAllowLoopback"] = conf.OAuthServer.AllowLoopback
	}

	return out
}
