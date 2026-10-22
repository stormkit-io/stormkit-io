package authwallhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type AuthCreateRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Metadata string `json:"metadata"`
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

	password, msg := loginInput{}.password(data.Password)

	if msg != "" {
		return shttp.BadRequest(map[string]any{
			"error": msg,
		})
	}

	data.Password = password

	if msg := authwall.ValidateMetadata(data.Metadata); msg != "" {
		return shttp.BadRequest(map[string]any{
			"error": msg,
		})
	}

	store := authwall.Store()
	teamID, err := store.EnvTeamID(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	if teamID == 0 {
		return shttp.BadRequest(map[string]any{
			"error": "The app does not belong to a team. Move it to a team first.",
		})
	}

	aw := &authwall.AuthWall{
		TeamID:        teamID,
		LoginEmail:    data.Email,
		LoginPassword: data.Password,
		Metadata:      data.Metadata,
		AppIDs:        []types.ID{req.App.ID},
	}

	if err := store.CreateLogin(req.Context(), aw); err != nil {
		if database.IsDuplicate(err) {
			return &shttp.Response{
				Status: http.StatusConflict,
				Data: map[string]string{
					"error": "A login with the same email already exists in this team.",
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
