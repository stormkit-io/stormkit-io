package volumeshandlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func HandlerVolumesDownloadURL(req *app.RequestContext) *shttp.Response {
	fileId := utils.StringToID(req.Query().Get("fileId"))

	if fileId == 0 {
		return shttp.BadRequest()
	}

	token, err := user.JWT(jwt.MapClaims{
		"token":  strings.Replace(req.Header.Get("Authorization"), "Bearer ", "", 1),
		"appId":  req.App.ID.String(),
		"envId":  req.EnvID.String(),
		"fileId": fileId.String(),
	})

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]string{
			"downloadUrl": admin.MustConfig().ApiURL(fmt.Sprintf("/volumes/download?token=%s", token)),
		},
	}
}
