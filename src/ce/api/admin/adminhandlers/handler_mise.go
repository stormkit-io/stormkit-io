package adminhandlers

import (
	"fmt"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/mise"
)

func handlerMise(req *user.RequestContext) *shttp.Response {
	version, err := mise.Client().Version()

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("error while fetching mise version: %s", err.Error()))
	}

	services := []string{
		rediscache.ServiceHosting,
		rediscache.ServiceWorkerserver,
	}

	status, err := rediscache.Status(req.Context(), "mise_update", services)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("error while fetching mise status: %s", err.Error()))
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"version": version,
			"status":  status,
		},
	}
}
