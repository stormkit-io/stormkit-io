package authwallhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type teamLoginCreateRequest struct {
	TeamID   types.ID   `json:"teamId"`
	Email    string     `json:"email"`
	Password string     `json:"password"`
	Metadata string     `json:"metadata"`
	AllApps  bool       `json:"allApps"`
	AppIDs   []types.ID `json:"appIds"`
}

// handlerTeamLoginCreate creates an Auth Wall login in the team.
func handlerTeamLoginCreate(req *user.RequestContext) *shttp.Response {
	data := &teamLoginCreateRequest{}

	if err := req.Post(data); err != nil {
		return shttp.Error(err)
	}

	t, res := teamAccess{req: req, manage: true}.team(data.TeamID)

	if res != nil {
		return res
	}

	if !utils.IsValidEmail(data.Email) {
		return shttp.BadRequest(map[string]any{
			"error": "Email is invalid.",
		})
	}

	input := loginInput{teamID: t.ID}
	password, msg := input.password(data.Password)

	if msg == "" {
		msg = authwall.ValidateMetadata(data.Metadata)
	}

	if msg != "" {
		return shttp.BadRequest(map[string]any{
			"error": msg,
		})
	}

	appIDs, msg, err := input.apps(req.Context(), data.AllApps, data.AppIDs)

	if err != nil {
		return shttp.Error(err)
	}

	if msg != "" {
		return shttp.BadRequest(map[string]any{
			"error": msg,
		})
	}

	aw := &authwall.AuthWall{
		TeamID:        t.ID,
		LoginEmail:    data.Email,
		LoginPassword: password,
		Metadata:      data.Metadata,
		AllApps:       data.AllApps,
		AppIDs:        appIDs,
	}

	if err := authwall.Store().CreateLogin(req.Context(), aw); err != nil {
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
		err := audit.FromRequestContext(req).
			WithAction(audit.CreateAction, audit.TypeAuthWall).
			WithDiff(&audit.Diff{
				New: audit.DiffFields{
					AuthWallCreateLoginEmail: aw.LoginEmail,
					AuthWallCreateLoginID:    aw.LoginID.String(),
					AuthWallLoginApps:        input.appsDiff(aw),
				},
			}).
			WithTeamID(t.ID).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return &shttp.Response{
		Status: http.StatusCreated,
		Data: map[string]any{
			"login": input.json(aw),
		},
	}
}
