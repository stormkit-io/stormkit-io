package publicapiv1

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// adaptAppHandler adapts a handler that expects *app.RequestContext to one that
// expects *RequestContext, bridging sub-package handlers to the local WithAPIKey pattern.
func adaptAppHandler(h func(*app.RequestContext) *shttp.Response) func(*RequestContext) *shttp.Response {
	return func(req *RequestContext) *shttp.Response {
		var envID types.ID
		if req.Env != nil {
			envID = req.Env.ID
		}

		return h(&app.RequestContext{
			RequestContext: &user.RequestContext{
				RequestContext: req.RequestContext,
			},
			App:   req.App,
			EnvID: envID,
			Token: req.Token,
		})
	}
}
