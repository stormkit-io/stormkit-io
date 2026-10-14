package apikeyhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// keyAuditor records the creation and removal of API keys in the audit log,
// including whether the key was read-only.
type keyAuditor struct {
	req *user.RequestContext
	key *apikey.Token
}

// log writes the entry for the given action. It runs after the key was
// already stored or removed, so a failure is logged rather than returned: the
// caller must not report a change that happened as failed. It is a no-op
// outside the enterprise edition, and for personal keys, which belong to no
// team and so would never appear in an audit view.
func (a *keyAuditor) log(action string) {
	if !a.req.License().IsEnterprise() {
		return
	}

	teamID, err := a.teamID()

	if err != nil {
		slog.Errorf("failed resolving team for api key audit: %s", err.Error())
		return
	}

	if teamID == 0 {
		return
	}

	fields := audit.DiffFields{
		APIKeyName:     a.key.Name,
		APIKeyScope:    a.key.Scope,
		APIKeyReadOnly: audit.Bool(a.key.ReadOnly),
	}

	diff := &audit.Diff{}

	if action == audit.DeleteAction {
		diff.Old = fields
	} else {
		diff.New = fields
	}

	err = audit.FromRequestContext(a.req).
		WithAction(action, audit.TypeAPIKey).
		WithDiff(diff).
		WithTeamID(teamID).
		WithAppID(a.key.AppID).
		WithEnvID(a.key.EnvID).
		Insert()

	if err != nil {
		slog.Errorf("failed writing api key audit: %s", err.Error())
	}
}

// teamID resolves the team the key belongs to, so the entry shows up in that
// team's audit log. Personal keys belong to no team.
func (a *keyAuditor) teamID() (types.ID, error) {
	if a.key.TeamID != 0 {
		return a.key.TeamID, nil
	}

	if a.key.AppID == 0 {
		return 0, nil
	}

	myApp, err := app.NewStore().AppByID(a.req.Context(), a.key.AppID)

	if err != nil || myApp == nil {
		return 0, err
	}

	return myApp.TeamID, nil
}
