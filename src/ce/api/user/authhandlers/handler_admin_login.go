package authhandlers

import (
	"net/http"
	"strings"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerAdminLogin(req *shttp.RequestContext) *shttp.Response {
	if res := hasAnyProviderEnabled(); res != nil {
		return res
	}

	data := AdminLoginRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	if res := validateAdminLoginRequest(data); res != nil {
		return res
	}

	store := user.NewStore()
	usr, err := store.UserByEmail(req.Context(), []string{data.Email})

	if err != nil {
		return shttp.Error(err)
	}

	if usr == nil {
		return shttp.NotAllowed()
	}

	cfg, err := admin.Store().Config(req.Context())

	if err != nil {
		return shttp.Error(err)
	}

	if cfg.AdminUserConfig == nil {
		return shttp.NotAllowed()
	}

	if !strings.EqualFold(cfg.AdminUserConfig.Email, data.Email) {
		return shttp.NotAllowed()
	}

	if utils.DecryptToString(cfg.AdminUserConfig.Password) != data.Password {
		return shttp.NotAllowed()
	}

	if err := user.NewStore().UpdateLastLogin(req.Context(), usr.ID); err != nil {
		return errorResponse(err, 0)
	}

	jwt, err := user.JWT(jwt.MapClaims{
		"uid": usr.ID.String(),
	})

	// Creating new token failed
	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"user":         usr.JSON(),
			"sessionToken": jwt,
		},
	}
}
