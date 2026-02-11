package adminhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type MiseUpdateRequest struct {
	Abort bool `json:"abort"`
}

func handlerMiseUpdate(req *user.RequestContext) *shttp.Response {
	services := []string{
		rediscache.ServiceHosting,
		rediscache.ServiceWorkerserver,
	}

	data := MiseUpdateRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	if data.Abort {
		if err := rediscache.DelAll("mise_update", services); err != nil {
			return shttp.Error(err)
		}

		return &shttp.Response{
			Status: http.StatusOK,
		}
	}

	if err := rediscache.SetAll("mise_update", rediscache.StatusSent, services); err != nil {
		return shttp.Error(err)
	}

	if err := rediscache.Broadcast(rediscache.EventMiseUpdate); err != nil {
		if err := rediscache.SetAll("mise_update", rediscache.StatusErr, services); err != nil {
			return shttp.Error(err)
		}

		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
	}
}
