package volumeshandlers

import (
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func HandlerVolumesDel(req *app.RequestContext) *shttp.Response {
	ctx := req.Context()
	ids := strings.Split(req.Query().Get("ids"), ",")

	if req.Query().Get("id") != "" {
		ids = []string{req.Query().Get("id")}
	}

	idsLen := len(ids)

	if idsLen > 100 || idsLen == 0 {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "Number of ids must be between 0 and 100.",
			},
		}
	}

	fileIds := []types.ID{}

	for _, id := range ids {
		fileIds = append(fileIds, utils.StringToID(id))
	}

	config, err := admin.Store().Config(ctx)

	if err != nil {
		return shttp.Error(err)
	}

	if config.VolumesConfig == nil {
		return volumesNotConfigured()
	}

	store := volumes.Store()
	files, err := store.SelectFiles(ctx, volumes.SelectFilesArgs{
		EnvID:  req.EnvID,
		FileID: fileIds,
	})

	if err != nil {
		return shttp.Error(err)
	}

	removedFiles, err := volumes.RemoveFiles(config.VolumesConfig, files)

	if err != nil {
		return shttp.Error(err)
	}

	if err = store.RemoveFiles(ctx, removedFiles, req.EnvID); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"removed": toJSON(removedFiles),
		},
	}
}
