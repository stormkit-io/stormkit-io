package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerAppConf(req *RequestContext) *shttp.Response {
	hostName := req.Query().Get("hostName")
	matches, err := appconf.FetchConfig(hostName)

	if err != nil {
		return shttp.Error(err)
	}

	// The host name is chosen by the caller, so only configs of the caller's
	// app, and of the key's environment for environment-level keys, are
	// returned. Configs carry the deployment's environment variables.
	configs := []*appconf.Config{}

	for _, cnf := range matches {
		if cnf.AppID != req.App.ID {
			continue
		}

		if req.Token.EnvID != 0 && cnf.EnvID != req.Token.EnvID {
			continue
		}

		configs = append(configs, cnf)
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
