package publicapiv1

import (
	"net/http"

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

	prv, err := skauth.NewStore().Provider(req.Context(), envID, provider)

	if err != nil {
		return shttp.Error(err)
	}

	if prv == nil || !prv.Status {
		return shttp.NotFound()
	}

	url, err := prv.Client().AuthCodeURL(skauth.AuthCodeURLParams{
		EnvID:        envID,
		ProviderName: provider,
	})

	if err != nil {
		return shttp.Error(err)
	}

	req.Redirect(url, http.StatusFound)

	return nil
}
