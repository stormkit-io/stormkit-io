package volumeshandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func HandlerVolumesSize(req *app.RequestContext) *shttp.Response {
	size, err := volumes.Store().VolumeSize(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]int64{
			"size": size,
		},
	}
}
