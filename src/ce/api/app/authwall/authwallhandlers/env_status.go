package authwallhandlers

import (
	"context"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// envStatus switches the wall of an environment on or off.
type envStatus struct {
	ctx context.Context

	// audit records the change. Nil when changes are not audited.
	audit *audit.Audit
}

// set sets the mode of the environment's wall, or returns why it cannot.
func (e envStatus) set(envID types.ID, status string) *shttp.Response {
	options := []string{
		authwall.StatusAll,
		authwall.StatusDev,
		authwall.StatusDisabled,
	}

	if !utils.InSliceString(options, status) {
		return shttp.BadRequest(map[string]any{
			"error": "Invalid authwall status. Available options are: all | dev | ''",
		})
	}

	store := authwall.Store()
	current, err := store.AuthWallConfig(e.ctx, envID)

	if err != nil {
		return shttp.Error(err)
	}

	if err := store.SetAuthWallConfig(e.ctx, envID, &authwall.Config{Status: status}); err != nil {
		return shttp.Error(err)
	}

	if err := appcache.Service().Reset(envID); err != nil {
		return shttp.Error(err)
	}

	if e.audit == nil {
		return nil
	}

	diff := &audit.Diff{
		Old: audit.DiffFields{AuthWallStatus: "off"},
		New: audit.DiffFields{AuthWallStatus: "off"},
	}

	if current != nil && current.Status != "" {
		diff.Old.AuthWallStatus = current.Status
	}

	if status != "" {
		diff.New.AuthWallStatus = status
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
