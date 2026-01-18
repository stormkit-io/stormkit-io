package skauthhandlers

import (
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"golang.org/x/oauth2"
)

// handleAuthRedirect initiates the OAuth2 authentication process by redirecting the user to the provider's authorization URL.
// Example request: GET /auth/v1?provider=google&envId=1
func handlerAuthRedirect(req *shttp.RequestContext) *shttp.Response {
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

	config, err := skauth.NewStore().Config(req.Context(), skauth.ConfigArgs{
		EnvID:        envID,
		ProviderName: provider,
	})

	if err != nil {
		fmt.Println("CONFIG ERR", err)
		return shttp.Error(err)
	}

	if config == nil {
		return shttp.NotFound()
	}

	state, err := user.JWT(jwt.MapClaims{
		"eid": envID,
		"prv": provider,
	})

	if err != nil {
		return shttp.Error(err)
	}

	req.Redirect(config.AuthCodeURL(state, oauth2.ApprovalForce, oauth2.AccessTypeOffline), http.StatusFound)

	return nil
}
