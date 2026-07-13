package adminhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/mise"
)

func handlerRuntimes(req *user.RequestContext) *shttp.Response {
	vc, err := admin.Store().Config(req.Context())

	if err != nil {
		return shttp.Error(err)
	}

	output, err := mise.Client().ListGlobal(req.Context())

	if err != nil {
		return shttp.Error(err)
	}

	requestedRuntimes := []string{}

	if vc.SystemConfig != nil && vc.SystemConfig.Runtimes != nil {
		requestedRuntimes = vc.SystemConfig.Runtimes
	}

	services := []string{
		rediscache.ServiceHosting,
		rediscache.ServiceWorkerserver,
	}

	status, err := rediscache.Status(req.Context(), rediscache.KEY_RUNTIMES_STATUS, services)

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"runtimes":    requestedRuntimes,
			"installed":   output,
			"autoInstall": vc.SystemConfig == nil || vc.SystemConfig.AutoInstall,
			"status":      status,
		},
	}
}
