package adminhandlers

import (
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type InstallDependenciesRequest struct {
	Runtimes    []string `json:"runtimes"` // <name>@<version> format
	AutoInstall bool     `json:"autoInstall"`
}

func handlerRuntimesInstall(req *user.RequestContext) *shttp.Response {
	ctx := req.Context()
	data := &InstallDependenciesRequest{}

	if err := req.Post(data); err != nil {
		return shttp.Error(err)
	}

	vc, err := admin.Store().Config(ctx)

	if err != nil {
		return shttp.Error(err)
	}

	if vc.SystemConfig == nil {
		vc.SystemConfig = &admin.SystemConfig{}
	}

	vc.SystemConfig.AutoInstall = data.AutoInstall
	vc.SystemConfig.Runtimes = []string{}

	for _, runtime := range data.Runtimes {
		runtime = strings.ReplaceAll(runtime, "\\u", "")

		if runtime == "" {
			continue
		}

		vc.SystemConfig.Runtimes = append(vc.SystemConfig.Runtimes, runtime)
	}

	if err := admin.Store().UpsertConfig(ctx, vc); err != nil {
		return shttp.Error(err)
	}

	services := []string{
		rediscache.ServiceHosting,
		rediscache.ServiceWorkerserver,
	}

	if err := rediscache.SetAll(rediscache.KEY_RUNTIMES_STATUS, rediscache.StatusSent, services); err != nil {
		return shttp.Error(err)
	}

	if err := rediscache.Broadcast(rediscache.EventRuntimesInstall); err != nil {
		if err := rediscache.SetAll(rediscache.KEY_RUNTIMES_STATUS, rediscache.StatusErr, services); err != nil {
			return shttp.Error(err)
		}

		return shttp.Error(err)
	}

	return shttp.OK()
}
