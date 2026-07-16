package skauthhandlers

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// AuthConfigUpdateRequest is a patch of the Stormkit Auth configuration. Every
// field is optional: a nil pointer (or nil slice for AllowedOrigins) leaves the
// corresponding stored setting untouched, so a client that saves only some
// fields — an older UI, an API caller, or an MCP tool — must not silently
// disable a live OAuth server, clear its MCP configuration, or reset the
// success URL out from under it.
type AuthConfigUpdateRequest struct {
	SuccessURL     *string  `json:"successUrl"`
	TTL            *int     `json:"tokenTtl"`
	Status         *bool    `json:"status"`
	AllowedOrigins []string `json:"allowedOrigins"`

	OAuthServerEnabled *bool   `json:"oauthServerEnabled"`
	OAuthResourcePath  *string `json:"oauthResourcePath"`
	OAuthAllowLoopback *bool   `json:"oauthAllowLoopback"`
	CookieDomain       *string `json:"cookieDomain"`
	LoginURL           *string `json:"loginUrl"`
}

const (
	successURLHint = "Provide a relative URL such as: /success"
	originHint     = "Provide values like https://app.example.com"
	sessionHint    = "Cookie domain must be a bare host such as .example.com; login URL a relative path or absolute URL."
	oauthPathHint  = "Provide a relative path such as: /mcp"
)

// ConfigValidationError is a user-facing validation failure carrying an
// actionable hint. Handlers surface Message and Hint in the 400 response body.
type ConfigValidationError struct {
	Message string
	Hint    string
}

func (e *ConfigValidationError) Error() string { return e.Message }

// ApplyConfigUpdate validates data and merges its provided fields into conf in
// place. It performs no I/O so it can be shared by the dashboard handler, the
// public API handler, and the MCP tool. On validation failure it returns a
// *ConfigValidationError.
func ApplyConfigUpdate(conf *buildconf.SKAuthConf, data AuthConfigUpdateRequest) error {
	if data.SuccessURL != nil {
		successURL := *data.SuccessURL

		if successURL != "" {
			parsed, err := url.Parse(successURL)

			if err != nil {
				return &ConfigValidationError{
					Message: "Success URL format is not valid. Make sure to provide a relative URL.",
					Hint:    successURLHint,
				}
			}

			if parsed.IsAbs() {
				return &ConfigValidationError{
					Message: "Success URL is not a relative URL.",
					Hint:    successURLHint,
				}
			}

			// Normalize to a leading-slash path so it joins cleanly with
			// an origin in the cross-origin redirect path.
			if !strings.HasPrefix(successURL, "/") {
				successURL = "/" + successURL
			}
		}

		conf.SuccessURL = successURL
	}

	if data.TTL != nil {
		conf.TTL = *data.TTL
	}

	if data.Status != nil {
		conf.Status = *data.Status
	}

	if data.AllowedOrigins != nil {
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
				return &ConfigValidationError{
					Message: fmt.Sprintf("Allowed origin %q must be a scheme + host only (no path, query, fragment, or userinfo).", raw),
					Hint:    originHint,
				}
			}

			allowedOrigins = append(allowedOrigins, origin)
		}

		conf.AllowedOrigins = allowedOrigins
	}

	if err := applySessionConf(conf, data); err != nil {
		return &ConfigValidationError{Message: err.Error(), Hint: sessionHint}
	}

	if err := applyOAuthServerConf(conf, data); err != nil {
		return &ConfigValidationError{Message: err.Error(), Hint: oauthPathHint}
	}

	return nil
}

func handlerAuthConfigUpdate(req *app.RequestContext) *shttp.Response {
	data := AuthConfigUpdateRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err, fmt.Sprintf("error while unmarshaling auth config request: %s", err.Error()))
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

	if err := ApplyConfigUpdate(env.AuthConf, data); err != nil {
		var verr *ConfigValidationError

		if errors.As(err, &verr) {
			return shttp.BadRequest(map[string]any{"error": verr.Message, "hint": verr.Hint})
		}

		return shttp.Error(err)
	}

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

// applySessionConf merges the session cookie fields from data into conf,
// leaving any omitted field untouched. It validates the cookie domain and the
// login URL the OAuth /authorize endpoint delegates to.
func applySessionConf(conf *buildconf.SKAuthConf, data AuthConfigUpdateRequest) error {
	if data.CookieDomain != nil {
		domain := strings.TrimSpace(*data.CookieDomain)

		// A cookie Domain attribute is a bare host (optionally dot-prefixed for
		// subdomain sharing): no scheme, port, path, or whitespace.
		if domain != "" && (strings.ContainsAny(domain, " \t\r\n/:") || strings.Contains(domain, "..")) {
			return fmt.Errorf("cookie domain %q must be a bare host such as .example.com", domain)
		}

		conf.CookieDomain = domain
	}

	if data.LoginURL != nil {
		loginURL := strings.TrimSpace(*data.LoginURL)

		if loginURL != "" {
			parsed, err := url.Parse(loginURL)

			// The login URL is either a relative path on the AS origin (leading
			// "/") or an absolute http(s) URL on the app's login origin. Anything
			// else can't be redirected to safely.
			if err != nil {
				return fmt.Errorf("login URL %q is not a valid URL", loginURL)
			}

			if parsed.IsAbs() {
				if parsed.Scheme != "http" && parsed.Scheme != "https" {
					return fmt.Errorf("login URL %q must be an http(s) URL or a relative path", loginURL)
				}
			} else if !strings.HasPrefix(loginURL, "/") || strings.HasPrefix(loginURL, "//") {
				// A leading "//" is a scheme-relative reference to another host, not
				// a path on the AS origin. Reject it here rather than let it silently
				// become a broken same-host path when delegateToLogin prefixes the
				// issuer.
				return fmt.Errorf("login URL %q must be an absolute URL or a leading-slash path", loginURL)
			}
		}

		conf.LoginURL = loginURL
	}

	return nil
}

// applyOAuthServerConf merges the OAuth-server fields from data into conf,
// leaving any field the client omitted untouched. It validates and normalizes
// the MCP resource path to a leading-slash relative path.
func applyOAuthServerConf(conf *buildconf.SKAuthConf, data AuthConfigUpdateRequest) error {
	if data.OAuthServerEnabled == nil && data.OAuthResourcePath == nil && data.OAuthAllowLoopback == nil {
		return nil
	}

	if conf.OAuthServer == nil {
		conf.OAuthServer = &buildconf.OAuthServerConf{}
	}

	if data.OAuthServerEnabled != nil {
		conf.OAuthServer.Enabled = *data.OAuthServerEnabled
	}

	if data.OAuthAllowLoopback != nil {
		conf.OAuthServer.AllowLoopback = *data.OAuthAllowLoopback
	}

	if data.OAuthResourcePath != nil {
		path := strings.TrimSpace(*data.OAuthResourcePath)

		if path != "" {
			parsed, err := url.Parse(path)

			// The path must be a bare relative path: no scheme/host, no query or
			// fragment, and no whitespace. Anything else would be stored verbatim
			// and then never match the normalized well-known probe, silently
			// breaking the connector with no error surfaced later.
			if err != nil || parsed.IsAbs() || parsed.Host != "" ||
				parsed.RawQuery != "" || parsed.Fragment != "" ||
				strings.ContainsAny(path, " \t\r\n") {
				return fmt.Errorf("MCP resource path %q must be a plain relative path such as /mcp", path)
			}

			path = utils.TrimPath(path)
		}

		conf.OAuthServer.ResourcePath = path
	}

	return nil
}
