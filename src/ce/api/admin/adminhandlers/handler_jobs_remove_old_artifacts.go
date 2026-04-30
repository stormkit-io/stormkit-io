package adminhandlers

import (
	"context"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerJobsRemoveOldArtifacts(req *user.RequestContext) *shttp.Response {
	vc, err := admin.Store().Config(req.Context())

	if err != nil {
		return shttp.Error(err)
	}

	retentionDays := 30

	if vc.SystemConfig != nil && vc.SystemConfig.ArtifactRetentionDays > 0 {
		retentionDays = vc.SystemConfig.ArtifactRetentionDays
	}

	ctx := context.WithValue(req.Context(), jobs.KeyContextNumberOfDeploymentsToDelete{}, 50)
	ids, err := jobs.RemoveDeploymentArtifactsManually(ctx, retentionDays)

	if err != nil {
		return &shttp.Response{
			Status: http.StatusInternalServerError,
			Data: map[string]any{
				"error": err.Error(),
			},
		}
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"deleted": ids,
		},
	}
}
