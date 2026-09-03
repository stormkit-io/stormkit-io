package adminhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerDisk(req *user.RequestContext) *shttp.Response {
	reports, err := admin.DiskReports()

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"services": reports,
		},
	}
}
