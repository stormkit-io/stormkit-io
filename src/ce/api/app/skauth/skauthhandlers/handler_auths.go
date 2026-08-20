package skauthhandlers

import (
	"fmt"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerAuths(req *app.RequestContext) *shttp.Response {
	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("error while fetching environment for skauth: %s, envId: %s", err.Error(), req.EnvID.String()))
	}

	data, err := ProvidersJSON(req.Context(), env)

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{Data: data}
}
