package volumeshandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func HandlerVolumesGet(req *app.RequestContext) *shttp.Response {
	beforeId := utils.StringToID(req.Query().Get("beforeId"))
	selectArgs := volumes.SelectFilesArgs{EnvID: req.EnvID}

	if beforeId != 0 {
		selectArgs.BeforeID = beforeId
	}

	files, err := volumes.Store().SelectFiles(req.Context(), selectArgs)

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"files": toJSON(files),
		},
	}
}
