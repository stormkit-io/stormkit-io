package volumeshandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerVolumesConfigGet(req *user.RequestContext) *shttp.Response {
	cfg, err := admin.Store().Config(req.Context())

	if err != nil {
		return shttp.Error(err)
	}

	if req.User.IsAdmin {
		return &shttp.Response{
			Status: http.StatusOK,
			Data: map[string]any{
				"config": cfg.VolumesConfig,
			},
		}
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]bool{
			"config": cfg.VolumesConfig != nil,
		},
	}
}
