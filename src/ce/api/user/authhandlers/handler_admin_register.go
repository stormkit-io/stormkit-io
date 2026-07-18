package authhandlers

import (
	"fmt"
	"net/http"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type AdminLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func validateAdminLoginRequest(data AdminLoginRequest) *shttp.Response {
	if data.Email == "" || data.Password == "" {
		return shttp.BadRequest()
	}

	if !utils.IsValidEmail(data.Email) {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "Email is invalid.",
			},
		}
	}

	if len(data.Password) < 6 {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "Password must be at least 6 characters long.",
			},
		}
	}

	return nil
}

func handlerAdminRegister(req *shttp.RequestContext) *shttp.Response {
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

	if len(data.Password) < admin.MinAdminPasswordLength {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": fmt.Sprintf("Password must be at least %d characters long.", admin.MinAdminPasswordLength),
			},
		}
	}

	cfg, err := admin.Store().Config(req.Context())

	if err != nil {
		return shttp.Error(err)
	}

	if cfg.AdminUserConfig != nil {
		return &shttp.Response{
			Status: http.StatusConflict,
			Data: map[string]string{
				"error": "Admin user already exists.",
			},
		}
	}

	adminCfg, err := admin.NewAdminUserConfig(data.Email, data.Password)

	if err != nil {
		return shttp.Error(err)
	}

	cfg.AdminUserConfig = adminCfg

	if err := admin.Store().UpsertConfig(req.Context(), cfg); err != nil {
		return shttp.Error(err)
	}

	usr, err := user.NewStore().MustUser(oauth.NewAdminUser(data.Email))

	if err != nil {
		return shttp.Error(err)
	}

	jwt, err := user.JWT(jwt.MapClaims{
		"uid": usr.ID.String(),
	})

	if err != nil {
		return shttp.Error(err)
	}

	if err := user.NewStore().UpdateLastLogin(req.Context(), usr.ID); err != nil {
		return errorResponse(err, 0)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"user":         usr.JSON(),
			"sessionToken": jwt,
		},
	}
}
