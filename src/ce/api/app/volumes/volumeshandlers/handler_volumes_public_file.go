package volumeshandlers

import (
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// HandlerVolumesPublicFile serves a file with the `hash` token. This token
// is composed of the file_id and env_id, and is encrypted using utils.Encrypt.
// The file is not served if it's not public.
func HandlerVolumesPublicFile(req *shttp.RequestContext) *shttp.Response {
	hash := req.Vars()["hash"]

	if hash == "" {
		return shttp.NotFound()
	}

	decrypted := utils.DecryptToString(hash)

	if decrypted == "" {
		return shttp.NotFound()
	}

	pieces := strings.Split(decrypted, ":")

	if len(pieces) != 2 {
		return shttp.NotFound()
	}

	fileID := utils.StringToID(pieces[0])
	envID := utils.StringToID(pieces[1])

	fileInfo, err := volumes.Store().FileByID(req.Context(), fileID)

	if err != nil {
		return shttp.Error(err)
	}

	if fileInfo == nil || fileInfo.EnvID != envID || !fileInfo.IsPublic {
		return shttp.NotFound()
	}

	cfg, err := admin.Store().Config(req.Context())

	if err != nil {
		return shttp.Error(err)
	}

	if cfg.VolumesConfig == nil {
		return volumesNotConfigured()
	}

	file, err := volumes.Download(cfg.VolumesConfig, fileInfo)

	if err != nil {
		return shttp.Error(err)
	}

	if file == nil {
		return shttp.NotFound()
	}

	headers := make(http.Header)
	modTime := fileInfo.CreatedAt.Time

	if fileInfo.UpdatedAt.Valid {
		modTime = fileInfo.UpdatedAt.Time
	}

	return &shttp.Response{
		Status:  http.StatusOK,
		Headers: headers,
		ServeContent: &shttp.ServeContent{
			Content: file,
			Name:    fileInfo.Name,
			ModTime: modTime,
		},
	}
}
