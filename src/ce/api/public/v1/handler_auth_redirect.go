package publicapiv1

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// HandlerAuthRedirect initiates the OAuth2 authentication process by redirecting the user to the provider's authorization URL.
// Example request: GET /v1/auth?provider=google&envId=1
func HandlerAuthRedirect(req *shttp.RequestContext) *shttp.Response {
	provider := req.Query().Get("provider")
	envID := utils.StringToID(req.Query().Get("envId"))

	if !utils.InSliceString(skauth.Providers, provider) {
		return shttp.BadRequest(map[string]any{
			"error": "invalid query parameter: missing or invalid provider",
		})
	}

	if envID == 0 {
		return shttp.BadRequest(map[string]any{
			"error": "invalid query parameter: missing or invalid envId",
		})
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), envID)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get environment by ID %d", envID))
	}

	if env == nil || env.AuthConf == nil || !env.AuthConf.Status {
		return shttp.NotFound()
	}

	prv, err := skauth.NewStore().Provider(req.Context(), envID, provider)

	if err != nil {
		return shttp.Error(err)
	}

	if prv == nil || !prv.Status {
		return shttp.NotFound()
	}

	referrer, errRes := getReferrer(req, env)

	if errRes != nil {
		return errRes
	}

	url, err := prv.Client().AuthCodeURL(skauth.AuthCodeURLParams{
		EnvID:        envID,
		ProviderName: provider,
		Referrer:     referrer,
	})

	if err != nil {
		return shttp.Error(err)
	}

	req.Redirect(url, http.StatusFound)

	return nil
}

// getReferrer determines the referrer URL for the authentication flow.
// It first checks the "referrer" query parameter, then the Referer header,
// and finally falls back to the app's preview URL if neither is provided.
// It also validates that the referrer domain belongs to the environment.
func getReferrer(req *shttp.RequestContext, env *buildconf.Env) (string, *shttp.Response) {
	referrer := utils.GetString(req.Query().Get("referrer"), req.Referer())

	if referrer == "" {
		apl, err := app.NewStore().AppByEnvID(req.Context(), env.ID)

		if err != nil {
			return "", shttp.Error(err, fmt.Sprintf("failed to get app by env ID %d: %s", env.ID, err.Error()))
		}

		referrer = admin.MustConfig().PreviewURL(apl.DisplayName, env.Name)
	} else {
		parsed, err := url.ParseRequestURI(referrer)

		if err != nil {
			return "", shttp.BadRequest(map[string]any{
				"error": "Referer is not a valid URL",
				"hint":  "Make sure the URL is an absolute URL with a valid format, e.g., https://myapp.com",
			})
		}

		hostname := parsed.Hostname()
		referrer = utils.GetString(parsed.Scheme, "https") + "://" + hostname
		filters := appconf.ParseHost(hostname)

		belongs, err := appconf.NewStore().BelongsToEnv(req.Context(), env.ID, filters)

		if err != nil {
			return "", shttp.Error(err, fmt.Sprintf("failed to check if referrer belongs to env: %s, ref: %s", err.Error(), referrer))
		}

		if !belongs {
			return "", shttp.NotAllowed()
		}
	}

	return referrer, nil
}
