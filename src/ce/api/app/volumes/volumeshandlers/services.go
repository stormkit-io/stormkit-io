package volumeshandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services sets the handlers for this service.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	appOpts := &app.Opts{Env: true}

	s.NewEndpoint("/volumes").
		Middleware(volumes.LimitRequestBody()).
		Handler(shttp.MethodGet, "", app.WithApp(HandlerVolumesGet, appOpts)).
		Handler(shttp.MethodPost, "", app.WithApp(HandlerVolumesPost, appOpts)).
		Handler(shttp.MethodDelete, "", app.WithApp(HandlerVolumesDel, appOpts)).
		Handler(shttp.MethodGet, "/download", HandlerVolumesDownloadFile).
		Handler(shttp.MethodGet, "/download/url", app.WithApp(HandlerVolumesDownloadURL, appOpts)).
		Handler(shttp.MethodPost, "/visibility", app.WithApp(HandlerVolumesChangeVisibility, appOpts)).
		Handler(shttp.MethodGet, "/file/{hash}", HandlerVolumesPublicFile).
		Handler(shttp.MethodGet, "/size", app.WithApp(HandlerVolumesSize, appOpts)).
		Handler(shttp.MethodGet, "/config", user.WithAuth(handlerVolumesConfigGet)).
		Handler(shttp.MethodPost, "/config", user.WithAdmin(handlerVolumesConfigSet))

	return s
}

func toJSON(files []*volumes.File) []map[string]any {
	response := []map[string]any{}

	for _, file := range files {
		data := map[string]any{
			"id":        file.ID.String(),
			"size":      file.Size,
			"name":      file.Name,
			"isPublic":  file.IsPublic,
			"createdAt": file.CreatedAt.Unix(),
			"mountType": file.Metadata["mountType"],
		}

		if file.IsPublic {
			data["publicLink"] = file.PublicLink()
		}

		if file.UpdatedAt.Valid {
			data["updatedAt"] = file.UpdatedAt.Unix()
		}

		response = append(response, data)
	}

	return response
}

func volumesNotConfigured() *shttp.Response {
	return &shttp.Response{
		Status: http.StatusBadRequest,
		Data: map[string]string{
			"error": "Volumes is not yet configured.",
		},
	}
}
