package apphandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// handlerAppDelete is a route to delete the app and all of its artifacts.
// This route will mark the app as deleted. It is the duty of the batch job
// later to delete the app and the artifacts.
func handlerAppDelete(req *app.RequestContext) *shttp.Response {
	deleted, err := app.NewStore().MarkAsDeleted(req.Context(), req.App.ID)

	if err != nil {
		return shttp.Error(err)
	}

	if !deleted {
		return shttp.NotFound()
	}

	if req.License().IsEnterprise() {
		diff := &audit.Diff{
			Old: audit.DiffFields{
				AppName: req.App.DisplayName,
				AppRepo: req.App.Repo,
			},
		}

		err = audit.FromRequestContext(req).
			WithAction(audit.DeleteAction, audit.TypeApp).
			WithDiff(diff).
			WithAppID(types.ID(0)). // We need to overwrite this otherwise the log will be deleted
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return shttp.OK()
}
