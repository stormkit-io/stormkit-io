package authwallhandlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

const (
	passwordMinLength = 8

	// passwordMaxLength is the longest password bcrypt can hash.
	passwordMaxLength = 72
)

// loginInput validates the fields of a login a team admin submits.
type loginInput struct {
	teamID types.ID
}

// password returns the trimmed password, or why it is not accepted.
func (l loginInput) password(password string) (string, string) {
	password = strings.TrimSpace(password)

	if len(password) < passwordMinLength {
		return "", fmt.Sprintf("Password must be at least %d characters long.", passwordMinLength)
	}

	if len(password) > passwordMaxLength {
		return "", fmt.Sprintf("Password must be at most %d characters long.", passwordMaxLength)
	}

	return password, ""
}

// apps returns the unique apps a login opens, or why the selection is not
// accepted. A login that opens all apps has none listed.
func (l loginInput) apps(ctx context.Context, allApps bool, appIDs []types.ID) ([]types.ID, string, error) {
	if allApps {
		return []types.ID{}, "", nil
	}

	unique := []types.ID{}
	seen := map[types.ID]bool{}

	for _, id := range appIDs {
		if id != 0 && !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}

	if len(unique) == 0 {
		return nil, "Select at least one app, or all apps.", nil
	}

	ok, err := authwall.Store().AreTeamApps(ctx, l.teamID, unique)

	if err != nil {
		return nil, "", err
	}

	if !ok {
		return nil, "Apps must belong to the team.", nil
	}

	return unique, "", nil
}

// appsDiff describes the apps a login opens for the audit log.
func (l loginInput) appsDiff(aw *authwall.AuthWall) string {
	if aw.AllApps {
		return "all"
	}

	ids := []string{}

	for _, id := range aw.AppIDs {
		ids = append(ids, id.String())
	}

	return strings.Join(ids, ",")
}

// json returns the login for API responses.
func (l loginInput) json(aw *authwall.AuthWall) map[string]any {
	appIDs := []string{}

	for _, id := range aw.AppIDs {
		appIDs = append(appIDs, id.String())
	}

	return map[string]any{
		"id":        aw.LoginID.String(),
		"email":     aw.LoginEmail,
		"metadata":  aw.Metadata,
		"allApps":   aw.AllApps,
		"appIds":    appIDs,
		"lastLogin": aw.LastLogin.Unix(),
		"createdAt": aw.CreatedAt.Unix(),
	}
}
