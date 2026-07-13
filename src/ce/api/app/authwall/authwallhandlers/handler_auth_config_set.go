package authwallhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type AuthConfigSetRequest struct {
	AuthWall string `json:"authwall"`
}

func handlerAuthConfigSet(req *app.RequestContext) *shttp.Response {
	data := AuthConfigSetRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	cnf := &authwall.Config{
		Status: data.AuthWall,
	}

	availableOptions := []string{
		authwall.StatusAll,
		authwall.StatusDev,
		authwall.StatusDisabled,
	}

	if !utils.InSliceString(availableOptions, cnf.Status) {
		return shttp.BadRequest(map[string]any{
			"error": "Invalid authwall status. Available options are: all | dev | ''",
		})
	}

	store := authwall.Store()
	current, err := store.AuthWallConfig(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	err = store.SetAuthWallConfig(req.Context(), req.EnvID, cnf)

	if err != nil {
		return shttp.Error(err)
	}

	if err := appcache.Service().Reset(req.EnvID); err != nil {
		return shttp.Error(err)
	}

	diff := &audit.Diff{
		Old: audit.DiffFields{
			AuthWallStatus: "off",
		},
		New: audit.DiffFields{
			AuthWallStatus: "off",
		},
	}

	if current != nil && current.Status != "" {
		diff.Old.AuthWallStatus = current.Status
	}

	if data.AuthWall != "" {
		diff.New.AuthWallStatus = data.AuthWall
	}

	if req.License().IsEnterprise() {
		err = audit.FromRequestContext(req).
			WithAction(audit.UpdateAction, audit.TypeAuthWall).
			WithDiff(diff).
			WithEnvID(req.EnvID).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return shttp.OK()
}
