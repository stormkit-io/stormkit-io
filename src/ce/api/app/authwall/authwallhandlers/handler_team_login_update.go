package authwallhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// teamLoginUpdateRequest holds the changes to a login. Absent fields are kept.
type teamLoginUpdateRequest struct {
	TeamID   types.ID    `json:"teamId"`
	ID       types.ID    `json:"id"`
	Password *string     `json:"password,omitempty"`
	Metadata *string     `json:"metadata,omitempty"`
	AllApps  *bool       `json:"allApps,omitempty"`
	AppIDs   *[]types.ID `json:"appIds,omitempty"`

	MagicLink *bool `json:"magicLink,omitempty"`
	// RemovePassword removes the login's password, so it signs in with a
	// magic link only.
	RemovePassword bool `json:"removePassword,omitempty"`
}

// handlerTeamLoginUpdate changes the password, magic link, metadata or app
// access of a team's Auth Wall login. Visitors already logged in keep their session, with
// the metadata it was issued with, until it expires.
func handlerTeamLoginUpdate(req *user.RequestContext) *shttp.Response {
	data := &teamLoginUpdateRequest{}

	if err := req.Post(data); err != nil {
		return shttp.Error(err)
	}

	t, res := teamAccess{req: req, manage: true}.team(data.TeamID)

	if res != nil {
		return res
	}

	store := authwall.Store()
	current, err := store.TeamLogin(req.Context(), t.ID, data.ID)

	if err != nil {
		return shttp.Error(err)
	}

	if current == nil {
		return shttp.NotFound()
	}

	input := loginInput{teamID: t.ID}
	updated := *current
	password := ""
	msg := ""

	if data.Password != nil && !data.RemovePassword {
		password, msg = input.password(*data.Password)
	}

	if data.MagicLink != nil {
		updated.MagicLink = *data.MagicLink
	}

	updated.HasPassword = password != "" || (current.HasPassword && !data.RemovePassword)

	if msg == "" {
		msg = input.signInMethods(updated.HasPassword, updated.MagicLink)
	}

	if data.Metadata != nil && msg == "" {
		updated.Metadata = *data.Metadata
		msg = authwall.ValidateMetadata(updated.Metadata)
	}

	if msg != "" {
		return shttp.BadRequest(map[string]any{
			"error": msg,
		})
	}

	replaceApps := data.AllApps != nil || data.AppIDs != nil

	if replaceApps {
		if data.AllApps != nil {
			updated.AllApps = *data.AllApps
		}

		if data.AppIDs != nil {
			updated.AppIDs = *data.AppIDs

			// A list of apps narrows a login that opens all apps.
			if data.AllApps == nil {
				updated.AllApps = false
			}
		}

		updated.AppIDs, msg, err = input.apps(req.Context(), updated.AllApps, updated.AppIDs)

		if err != nil {
			return shttp.Error(err)
		}

		if msg != "" {
			return shttp.BadRequest(map[string]any{
				"error": msg,
			})
		}
	}

	found, err := store.UpdateLogin(req.Context(), authwall.UpdateLoginParams{
		Login:          &updated,
		Password:       password,
		RemovePassword: data.RemovePassword,
		ReplaceApps:    replaceApps,
	})

	if err != nil {
		return shttp.Error(err)
	}

	if !found {
		return shttp.NotFound()
	}

	if req.License().IsEnterprise() {
		diff := &audit.Diff{
			Old: audit.DiffFields{
				AuthWallLoginEmail:     current.LoginEmail,
				AuthWallLoginApps:      input.appsDiff(current),
				AuthWallLoginMetadata:  current.Metadata,
				AuthWallLoginMagicLink: &current.MagicLink,
			},
			New: audit.DiffFields{
				AuthWallLoginEmail:     updated.LoginEmail,
				AuthWallLoginApps:      input.appsDiff(&updated),
				AuthWallLoginMetadata:  updated.Metadata,
				AuthWallLoginMagicLink: &updated.MagicLink,
			},
		}

		if password != "" || (data.RemovePassword && current.HasPassword) {
			changed := true
			diff.New.AuthWallPasswordChanged = &changed
		}

		err := audit.FromRequestContext(req).
			WithAction(audit.UpdateAction, audit.TypeAuthWallLogin).
			WithDiff(diff).
			WithTeamID(t.ID).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"login": input.json(&updated),
		},
	}
}
