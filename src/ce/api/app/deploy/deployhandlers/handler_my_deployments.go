package deployhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerMyDeployments(req *user.RequestContext) *shttp.Response {
	userID := req.User.ID
	teamID := utils.StringToID(req.Query().Get("teamId"))
	deploymentID := utils.StringToID(req.Query().Get("deploymentId"))
	includeLogs := false

	if deploymentID != 0 {
		includeLogs = true

		// Allow admins see deployment logs when the link is shared
		if req.User.IsAdmin {
			userID = 0
		}
	}

	deployments, err := deploy.
		NewStore().
		MyDeployments(req.Context(), &deploy.DeploymentsQueryFilters{
			UserID:       userID,
			TeamID:       teamID,
			DeploymentID: deploymentID,
			EnvID:        utils.StringToID(req.Query().Get("envId")),
			IncludeLogs:  &includeLogs,
		})

	if err != nil {
		return shttp.Error(err)
	}

	response := []map[string]any{}

	for _, d := range deployments {
		response = append(response, d.JSON(includeLogs))
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"deployments": response,
		},
	}
}
