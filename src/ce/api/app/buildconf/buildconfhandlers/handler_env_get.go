package buildconfhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerEnv retrieves a configuration for the given app and environment.
func handlerEnv(req *app.RequestContext) *shttp.Response {
	env := req.Vars()["env"]
	conf, err := buildconf.NewStore().Environment(req.Context(), req.App.ID, env)

	if err != nil {
		return shttp.Error(err)
	}

	if conf == nil {
		return shttp.NotFound()
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   map[string]*buildconf.Env{"config": conf},
	}
}
