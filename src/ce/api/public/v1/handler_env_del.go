package publicapiv1

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerEnvDel(req *RequestContext) *shttp.Response {
	deleted, err := buildconf.NewStore().MarkAsDeleted(req.Context(), req.Env.ID)

	if err != nil {
		return shttp.Error(err)
	}

	if !deleted {
		return shttp.NotFound()
	}

	if req.License().IsEnterprise() {
		err := audit.FromRequestContext(req).
			WithAction(audit.DeleteAction, audit.TypeEnv).
			WithDiff(&audit.Diff{Old: audit.DiffFields{EnvName: req.Env.Name}}).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return shttp.OK()
}
