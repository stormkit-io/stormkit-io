package volumeshandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func HandlerVolumesDownloadFile(req *shttp.RequestContext) *shttp.Response {
	token := req.Query().Get("token")

	if token == "" {
		return shttp.BadRequest()
	}

	jwt := user.ParseJWT(&user.ParseJWTArgs{
		Bearer:  token,
		MaxMins: 5,
	})

	for _, arg := range []string{"fileId", "appId", "envId"} {
		if _, ok := jwt[arg].(string); !ok {
			return shttp.NotAllowed()
		}
	}

	fileID := utils.StringToID(jwt["fileId"].(string))
	envID := utils.StringToID(jwt["envId"].(string))

	fileInfo, err := volumes.Store().FileByID(req.Context(), fileID)

	if err != nil {
		return shttp.Error(err)
	}

	if fileInfo.EnvID != envID {
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
	headers.Set("Content-Disposition", "attachment; filename="+fileInfo.Name)

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
