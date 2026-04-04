package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerEnvList(req *RequestContext) *shttp.Response {
	envs, err := buildconf.NewStore().ListEnvironments(req.Context(), req.App.ID)

	if err != nil {
		return shttp.Error(err)
	}

	if envs == nil {
		envs = []*buildconf.Env{}
	}

	cfg := admin.MustConfig()

	for _, env := range envs {
		env.Preview = cfg.PreviewURL(req.App.DisplayName, env.Name)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   map[string]any{"environments": envs},
	}
}
