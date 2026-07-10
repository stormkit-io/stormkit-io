package maintenancehandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/maintenance"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerConfigGet(req *app.RequestContext) *shttp.Response {
	cnf, err := maintenance.Store().Config(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"maintenance": cnf.Status,
		},
	}
}
