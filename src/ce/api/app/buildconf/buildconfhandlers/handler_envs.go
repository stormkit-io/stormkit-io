package buildconfhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type HandlerEnvsResponse struct {
	Envs []*buildconf.Env `json:"envs"`
}

// handlerEnvs retrieves list of environments.
// Diferrence between this handler and the overview is that
// overview returns also the deployments.
func handlerEnvs(req *app.RequestContext) *shttp.Response {
	confs, err := buildconf.NewStore().ListEnvironments(req.Context(), req.App.ID)

	if err != nil {
		return shttp.Error(err)
	}

	if confs == nil {
		return &shttp.Response{
			Data: map[string]any{
				"envs": []*buildconf.Env{},
			},
		}
	}

	cfg := admin.MustConfig()

	for _, conf := range confs {
		conf.Preview = cfg.PreviewURL(req.App.DisplayName, conf.Name)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: &HandlerEnvsResponse{
			Envs: confs,
		},
	}
}
