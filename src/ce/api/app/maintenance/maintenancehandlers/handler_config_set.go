package maintenancehandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/maintenance"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type ConfigSetRequest struct {
	Maintenance string `json:"maintenance"`
}

func handlerConfigSet(req *app.RequestContext) *shttp.Response {
	data := ConfigSetRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	cnf := &maintenance.Config{
		Status: data.Maintenance,
	}

	availableOptions := []string{
		maintenance.StatusOn,
		maintenance.StatusDisabled,
	}

	if !utils.InSliceString(availableOptions, cnf.Status) {
		return shttp.BadRequest(map[string]any{
			"error": "Invalid maintenance status. Available options are: on | ''",
		})
	}

	store := maintenance.Store()
	current, err := store.Config(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	if err := store.SetConfig(req.Context(), req.EnvID, cnf); err != nil {
		return shttp.Error(err)
	}

	if err := appcache.Service().Reset(req.EnvID); err != nil {
		return shttp.Error(err)
	}

	diff := &audit.Diff{
		Old: audit.DiffFields{
			MaintenanceStatus: "off",
		},
		New: audit.DiffFields{
			MaintenanceStatus: "off",
		},
	}

	if current != nil && current.Status != "" {
		diff.Old.MaintenanceStatus = current.Status
	}

	if data.Maintenance != "" {
		diff.New.MaintenanceStatus = data.Maintenance
	}

	if req.License().IsEnterprise() {
		err = audit.FromRequestContext(req).
			WithAction(audit.UpdateAction, audit.TypeMaintenance).
			WithDiff(diff).
			WithEnvID(req.EnvID).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return shttp.OK()
}
