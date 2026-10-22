package authwallhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type AuthConfigSetRequest struct {
	AuthWall string `json:"authwall"`
}

func handlerAuthConfigSet(req *app.RequestContext) *shttp.Response {
	data := AuthConfigSetRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	update := envStatus{ctx: req.Context()}

	if req.License().IsEnterprise() {
		update.audit = audit.FromRequestContext(req)
	}

	if res := update.set(req.EnvID, data.AuthWall); res != nil {
		return res
	}

	return shttp.OK()
}
