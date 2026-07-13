package volumeshandlers

import (
	"mime"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func HandlerVolumesPost(req *app.RequestContext) *shttp.Response {
	if req.MultipartForm == nil {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "Invalid request: expected multipart/form-data with files under the \"files\" field.",
			},
		}
	}

	cfg, err := admin.Store().Config(req.Context())

	if err != nil {
		return nil
	}

	if cfg.VolumesConfig == nil {
		return volumesNotConfigured()
	}

	files := req.MultipartForm.File["files"]

	uploadedFiles := []*volumes.File{}
	uploadedFilesKeys := map[string]*volumes.File{}
	failedFiles := map[string]string{}

	for _, fileHeader := range files {
		_, params, err := mime.ParseMediaType(fileHeader.Header.Get("Content-Disposition"))

		if err != nil {
			slog.Errorf("cannot parse content-disposition: %s", err.Error())
		}

		if params == nil {
			params = map[string]string{}
		}

		file, err := volumes.Upload(cfg.VolumesConfig, volumes.UploadArgs{
			AppID:              req.App.ID,
			EnvID:              req.EnvID,
			FileHeader:         volumes.FromFileHeader(fileHeader),
			ContentDisposition: params,
		})

		if err != nil {
			failedFiles[utils.GetString(params["filename"], fileHeader.Filename)] = err.Error()
			continue
		}

		// Prevents duplicate files
		if uploadedFilesKeys[file.Name] != nil {
			uploadedFilesKeys[file.Name].Size = file.Size
			uploadedFilesKeys[file.Name].CreatedAt = file.CreatedAt
			continue
		}

		uploadedFiles = append(uploadedFiles, file)
		uploadedFilesKeys[file.Name] = file
	}

	if len(uploadedFiles) > 0 {
		if err := volumes.Store().Insert(req.Context(), uploadedFiles, req.EnvID); err != nil {
			return shttp.Error(err)
		}
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"files":  toJSON(uploadedFiles),
			"failed": failedFiles,
		},
	}
}
