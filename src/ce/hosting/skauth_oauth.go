package hosting

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// oauthProviders is the set of OAuth2 providers served from the hosting layer
// at /_stormkit/auth/<provider>. Email and magic-link have their own paths.
var oauthProviders = map[string]bool{
	skauth.ProviderGoogle: true,
	skauth.ProviderX:      true,
}

// handleOAuthInitiate starts the OAuth2 authorization flow for providerName.
// The environment is derived from the request Host (no envId query param), and
// the post-login redirect target is resolved from the request (see
// resolveRedirect). On success it 302-redirects to the provider's consent
// screen; the provider then calls back the central /v1/auth/callback endpoint.
func (m *skAuthMiddleware) handleOAuthInitiate(providerName string) (*shttp.Response, error) {
	if m.req.Method != http.MethodGet {
		return &shttp.Response{
			Status: http.StatusMethodNotAllowed,
			Data:   map[string]any{"errors": []string{"method not allowed"}},
		}, nil
	}

	req := m.req.RequestContext
	envID := m.req.Host.Config.EnvID

	if envID == 0 {
		return shttp.NotFound(), nil
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), envID)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get environment by ID %d", envID)), nil
	}

	if env == nil || env.AuthConf == nil || !env.AuthConf.Status {
		return shttp.NotFound(), nil
	}

	prv, err := skauth.NewStore().Provider(req.Context(), envID, providerName)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get provider %s for env %d", providerName, envID)), nil
	}

	if prv == nil || !prv.Status {
		return shttp.NotFound(), nil
	}

	redirect, resp := m.resolveRedirect(env)

	if resp != nil {
		return resp, nil
	}

	authURL, err := prv.Client().AuthCodeURL(skauth.AuthCodeURLParams{
		EnvID:        envID,
		ProviderName: providerName,
		Referrer:     redirect,
	})

	if err != nil {
		return shttp.Error(err, "failed to build authorization URL"), nil
	}

	return &shttp.Response{
		Status:   http.StatusFound,
		Redirect: &authURL,
	}, nil
}

// resolveRedirect determines and validates the post-login redirect origin.
// Precedence: ?redirect= query param → Origin header → Referer header. When the
// environment configures an AllowedOrigins allow-list, a cross-origin target
// must be on it (otherwise 403). When no target is supplied — or none is
// allow-listed — the flow falls back to the app's own origin (the single-host
// case). The returned value is always scheme + host with no path.
func (m *skAuthMiddleware) resolveRedirect(env *buildconf.Env) (string, *shttp.Response) {
	req := m.req.RequestContext

	candidate := utils.GetString(
		req.Query().Get("redirect"),
		m.req.Header.Get("Origin"),
		m.req.Header.Get("Referer"),
	)

	if candidate != "" {
		parsed, err := url.ParseRequestURI(candidate)

		if err != nil {
			return "", shttp.BadRequest(map[string]any{
				"errors": []string{"redirect is not a valid absolute URL"},
			})
		}

		candidate = utils.GetString(parsed.Scheme, "https") + "://" + parsed.Host
	}

	// The app's own origin — the fallback whenever no allow-listed cross-origin
	// target applies. Hosting always serves over HTTPS in production.
	own := "https://" + m.req.Host.Name

	if candidate == "" {
		return own, nil
	}

	if len(env.AuthConf.AllowedOrigins) > 0 {
		if !env.AuthConf.IsAllowedOrigin(candidate) {
			return "", &shttp.Response{
				Status: http.StatusForbidden,
				Data:   map[string]any{"errors": []string{"redirect origin is not allowed"}},
			}
		}

		return candidate, nil
	}

	// No allow-list configured: ignore the cross-origin target and stay
	// single-host so the session lands back on this domain.
	return own, nil
}

// isOAuthProviderPath reports whether path is /_stormkit/auth/<oauth-provider>
// and returns the provider name when it is.
func isOAuthProviderPath(path string) (string, bool) {
	provider, ok := strings.CutPrefix(path, "/_stormkit/auth/")

	if !ok || !oauthProviders[provider] {
		return "", false
	}

	return provider, true
}
