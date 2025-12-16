package deployhandlers

import (
	"fmt"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
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
		response = append(response, jsonResponse(d, includeLogs))
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"deployments": response,
		},
	}
}

func jsonResponse(d *deploy.Deployment, withLogs bool) map[string]any {
	appID := d.AppID.String()
	envID := d.EnvID.String()
	depID := d.ID.String()
	var deploymentLogs []*deploy.Log
	var statusChecksLogs []*deploy.Log

	if withLogs {
		deploymentLogs = d.PrepareLogs(d.Logs.ValueOrZero(), false)
		statusChecksLogs = d.PrepareLogs(d.StatusChecks.ValueOrZero(), true)
	}

	jsonMap := map[string]any{
		"id":                 depID,
		"appId":              appID,
		"envId":              envID,
		"envName":            d.Env,
		"displayName":        d.DisplayName,
		"repo":               d.CheckoutRepo,
		"logs":               deploymentLogs,
		"branch":             d.Branch,
		"createdAt":          d.CreatedAt.UnixStr(),
		"stoppedAt":          d.StoppedAt.UnixStr(),
		"stoppedManually":    d.ExitCode.ValueOrZero() == -1,
		"status":             d.Status(),
		"snapshot":           d.Snapshot(),
		"error":              d.Error.ValueOrZero(),
		"isAutoDeploy":       d.IsAutoDeploy,
		"isAutoPublish":      d.ShouldPublish,
		"previewUrl":         admin.MustConfig().PreviewURL(d.DisplayName, d.ID.String()),
		"detailsUrl":         fmt.Sprintf("/apps/%s/environments/%s/deployments/%s", appID, envID, depID),
		"apiPathPrefix":      d.APIPathPrefix.ValueOrZero(),
		"statusChecks":       statusChecksLogs,
		"statusChecksPassed": d.StatusChecksPassed,
		"duration":           calculateDuration(d.CreatedAt, d.StoppedAt),
		"published":          []map[string]any{},
		"commit": map[string]any{
			"sha":     d.Commit.ID.ValueOrZero(),
			"author":  d.Commit.Author.ValueOrZero(),
			"message": d.Commit.Message.ValueOrZero(),
		},
	}

	if d.Published != nil {
		for _, p := range d.Published {
			jsonMap["published"] = append(jsonMap["published"].([]map[string]any), map[string]any{
				"envId":      p.EnvID.String(),
				"percentage": p.Percentage,
			})
		}
	}

	if d.UploadResult != nil {
		jsonMap["uploadResult"] = map[string]any{
			"clientBytes":     d.UploadResult.ClientBytes,
			"serverBytes":     d.UploadResult.ServerBytes,
			"serverlessBytes": d.UploadResult.ServerlessBytes,
		}
	}

	if !withLogs {
		jsonMap["logs"] = nil
		jsonMap["statusChecks"] = nil
	}

	return jsonMap
}

func calculateDuration(createdAt, stoppedAt utils.Unix) int64 {
	if stoppedAt.IsZero() {
		return 0
	}

	return stoppedAt.Unix() - createdAt.Unix()
}
