package adminhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerSystemSettings(req *user.RequestContext) *shttp.Response {
	vc, err := admin.Store().Config(req.Context())

	if err != nil {
		return shttp.Error(err)
	}

	artifactRetentionDays := 30

	if vc.SystemConfig != nil && vc.SystemConfig.ArtifactRetentionDays > 0 {
		artifactRetentionDays = vc.SystemConfig.ArtifactRetentionDays
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"artifactRetentionDays": artifactRetentionDays,
		},
	}
}
