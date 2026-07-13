package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerAppConf(req *RequestContext) *shttp.Response {
	hostName := req.Query().Get("hostName")
	configs, err := appconf.FetchConfig(hostName)

	if err != nil {
		return shttp.Error(err)
	}

	if len(configs) == 0 {
		return &shttp.Response{
			Status: http.StatusNoContent,
			Data: map[string]string{
				"error": "Config is not found. Did you publish your deployment?",
			},
		}
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"configs": configs,
		},
	}
}
