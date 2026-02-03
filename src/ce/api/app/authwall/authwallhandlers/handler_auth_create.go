package authwallhandlers

import (
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type AuthCreateRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func handlerAuthCreate(req *app.RequestContext) *shttp.Response {
	data := &AuthCreateRequest{}

	if err := req.Post(data); err != nil {
		return shttp.Error(err)
	}

	if !utils.IsValidEmail(data.Email) {
		return shttp.BadRequest(map[string]any{
			"error": "Email is invalid.",
		})
	}

	data.Password = strings.TrimSpace(data.Password)

	if len(data.Password) < 8 {
		return shttp.BadRequest(map[string]any{
			"error": "Password must be at least 8 characters long.",
		})
	}

	aw := &authwall.AuthWall{
		LoginEmail:    data.Email,
		LoginPassword: data.Password,
		EnvID:         req.EnvID,
	}

	err := authwall.Store().CreateLogin(req.Context(), aw)

	if err != nil {
		if database.IsDuplicate(err) {
			return &shttp.Response{
				Status: http.StatusConflict,
				Data: map[string]string{
					"error": "A user with the same email already exists for this environment.",
				},
			}
		}

		return shttp.Error(err)
	}

	if req.License().IsEnterprise() {
		diff := &audit.Diff{
			New: audit.DiffFields{
				AuthWallCreateLoginEmail: data.Email,
				AuthWallCreateLoginID:    aw.LoginID.String(),
			},
		}

		err = audit.FromRequestContext(req).
			WithAction(audit.CreateAction, audit.TypeAuthWall).
			WithDiff(diff).
			WithEnvID(req.EnvID).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return shttp.OK()
}
