package authwallhandlers

import (
	"context"
	"slices"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// envStatus changes the wall of an environment.
type envStatus struct {
	ctx context.Context

	// audit records the change. Nil when changes are not audited.
	audit *audit.Audit
}

// envChange is a change to an environment's wall. Nil fields are left as they
// are.
type envChange struct {
	Status      *string
	PublicPaths *[]string
}

// set sets the mode of the environment's wall, or returns why it cannot.
func (e envStatus) set(envID types.ID, status string) *shttp.Response {
	return e.update(envID, envChange{Status: &status})
}

// update applies the change to the environment's wall, or returns why it
// cannot.
func (e envStatus) update(envID types.ID, change envChange) *shttp.Response {
	options := []string{
		authwall.StatusAll,
		authwall.StatusDev,
		authwall.StatusDisabled,
	}

	if change.Status != nil && !utils.InSliceString(options, *change.Status) {
		return shttp.BadRequest(map[string]any{
			"error": "Invalid authwall status. Available options are: all | dev | ''",
		})
	}

	var publicPaths []string

	if change.PublicPaths != nil {
		paths, msg := authwall.ValidatePublicPaths(*change.PublicPaths)

		if msg != "" {
			return shttp.BadRequest(map[string]any{"error": msg})
		}

		publicPaths = paths
	}

	store := authwall.Store()
	current, err := store.AuthWallConfig(e.ctx, envID)

	if err != nil {
		return shttp.Error(err)
	}

	if current == nil {
		current = &authwall.Config{}
	}

	next := *current

	if change.Status != nil {
		next.Status = *change.Status
	}

	if change.PublicPaths != nil {
		next.PublicPaths = publicPaths
	}

	if err := store.SetAuthWallConfig(e.ctx, envID, &next); err != nil {
		return shttp.Error(err)
	}

	if err := appcache.Service().Reset(envID); err != nil {
		return shttp.Error(err)
	}

	if e.audit == nil {
		return nil
	}

	diff := &audit.Diff{
		Old: audit.DiffFields{AuthWallStatus: e.statusLabel(current.Status)},
		New: audit.DiffFields{AuthWallStatus: e.statusLabel(next.Status)},
	}

	if !slices.Equal(current.PublicPaths, next.PublicPaths) {
		diff.Old.AuthWallPublicPaths = current.PublicPaths
		diff.New.AuthWallPublicPaths = next.PublicPaths
	}

	err = e.audit.
		WithAction(audit.UpdateAction, audit.TypeAuthWall).
		WithDiff(diff).
		WithEnvID(envID).
		Insert()

	if err != nil {
		return shttp.Error(err)
	}

	return nil
}

// statusLabel is how the audit log names a wall's mode.
func (e envStatus) statusLabel(status string) string {
	if status == "" {
		return "off"
	}

	return status
}
