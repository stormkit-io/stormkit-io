package publicapiv1

import (
	"errors"
	"mime"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerVolumesPost(req *RequestContext) *shttp.Response {
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
		return shttp.Error(err)
	}

	if cfg.VolumesConfig == nil {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "File storage is not yet configured.",
			},
		}
	}

	files := req.MultipartForm.File["files"]

	if len(files) == 0 {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "At least one file is required. Send files under the \"files\" field.",
			},
		}
	}

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

		// Sanitization is handled inside the Upload method.
		fileName := utils.GetString(params["filename"], fileHeader.Filename)

		file, err := volumes.Upload(cfg.VolumesConfig, volumes.UploadArgs{
			AppID:              req.App.ID,
			EnvID:              req.Env.ID,
			FileHeader:         volumes.FromFileHeader(fileHeader),
			ContentDisposition: params,
		})

		if err != nil {
			failedFiles[fileName] = err.Error()
			continue
		}

		if file == nil {
			slog.Errorf("volumes.Upload returned nil file without error for app %s env %s", req.App.ID, req.Env.ID)
			return shttp.Error(errors.New("file storage is misconfigured: upload handler returned no file"))
		}

		// Deduplicate: keep the last version of a file uploaded in the same request.
		if uploadedFilesKeys[file.Name] != nil {
			uploadedFilesKeys[file.Name].Size = file.Size
			uploadedFilesKeys[file.Name].CreatedAt = file.CreatedAt
			continue
		}

		uploadedFiles = append(uploadedFiles, file)
		uploadedFilesKeys[file.Name] = file
	}

	if len(uploadedFiles) > 0 {
		if err := volumes.Store().Insert(req.Context(), uploadedFiles, req.Env.ID); err != nil {
			return shttp.Error(err)
		}
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"files":  filesToJSON(uploadedFiles),
			"failed": failedFiles,
		},
	}
}

func filesToJSON(files []*volumes.File) []map[string]any {
	result := make([]map[string]any, 0, len(files))

	for _, file := range files {
		data := map[string]any{
			"id":        file.ID.String(),
			"name":      file.Name,
			"size":      file.Size,
			"isPublic":  file.IsPublic,
			"createdAt": file.CreatedAt.Unix(),
		}

		if file.IsPublic {
			data["publicLink"] = file.PublicLink()
		}

		if file.UpdatedAt.Valid {
			data["updatedAt"] = file.UpdatedAt.Unix()
		}

		result = append(result, data)
	}

	return result
}
