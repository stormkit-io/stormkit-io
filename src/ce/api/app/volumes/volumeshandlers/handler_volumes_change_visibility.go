package volumeshandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type ChangeVisibilityRequest struct {
	Visibility string   `json:"visibility"` // public|private
	FileID     types.ID `json:"fileId"`
}

func HandlerVolumesChangeVisibility(req *app.RequestContext) *shttp.Response {
	data := ChangeVisibilityRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	if data.Visibility != "public" && data.Visibility != "private" {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "Invalid visibility provided: can be one of public or private.",
			},
		}
	}

	store := volumes.Store()
	file, err := store.FileByID(req.Context(), data.FileID)

	if err != nil {
		return shttp.Error(err)
	}

	if file == nil || file.EnvID != req.EnvID {
		return shttp.NotFound()
	}

	err = store.ChangeVisibility(req.Context(), file.ID, data.Visibility == "public")

	if err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
