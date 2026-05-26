package skauthhandlers

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type AuthConfigUpdateRequest struct {
	SuccessURL     string   `json:"successUrl"`
	TTL            int      `json:"tokenTtl"`
	Status         bool     `json:"status"`
	AllowedOrigins []string `json:"allowedOrigins"`
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

		// Normalize to a leading-slash path so it joins cleanly with
		// an origin in the cross-origin redirect path.
		if !strings.HasPrefix(data.SuccessURL, "/") {
			data.SuccessURL = "/" + data.SuccessURL
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

	allowedOrigins := make([]string, 0, len(data.AllowedOrigins))

	for _, raw := range data.AllowedOrigins {
		origin := strings.TrimSpace(strings.TrimRight(raw, "/"))

		if origin == "" {
			continue
		}

		parsed, perr := url.Parse(origin)

		// An allowed origin must be exactly scheme + host (no path, query,
		// fragment, or userinfo). Anything else can never match a browser
		// Origin header and would silently misconfigure the allow-list.
		if perr != nil || !parsed.IsAbs() || parsed.Host == "" ||
			parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
			return shttp.BadRequest(map[string]any{
				"error": fmt.Sprintf("Allowed origin %q must be a scheme + host only (no path, query, fragment, or userinfo).", raw),
				"hint":  "Provide values like https://app.example.com",
			})
		}

		allowedOrigins = append(allowedOrigins, origin)
	}

	env.AuthConf.SuccessURL = data.SuccessURL
	env.AuthConf.TTL = data.TTL
	env.AuthConf.Status = data.Status
	env.AuthConf.AllowedOrigins = allowedOrigins

	err = store.SaveAuthConf(req.Context(), req.EnvID, env.AuthConf)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("error while saving auth conf for env: %s, err=%s", req.EnvID.String(), err.Error()))
	}

	// Invalidate the hosting cache so the SKAuth middleware picks up the
	// new TTL/SuccessURL/AllowedOrigins on the next request. Without this
	// the cached appconf keeps serving the previous values until the cache
	// expires on its own, which manifests as users getting kicked out
	// before the new TTL has elapsed.
	if err := appcache.Service().Reset(req.EnvID); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
