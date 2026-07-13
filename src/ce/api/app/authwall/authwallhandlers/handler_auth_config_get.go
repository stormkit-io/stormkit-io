package authwallhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerAuthConfigGet(req *app.RequestContext) *shttp.Response {
	cnf, err := authwall.Store().AuthWallConfig(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"authwall": cnf.Status,
		},
	}
}
