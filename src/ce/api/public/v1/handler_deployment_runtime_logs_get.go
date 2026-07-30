package publicapiv1

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/applog"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// RuntimeLog is the API representation of a runtime log entry. IDs and the
// timestamp are serialized as strings to avoid precision loss in JavaScript.
type RuntimeLog struct {
	ID           string `json:"id"`
	AppID        string `json:"appId"`
	DeploymentID string `json:"deploymentId"`
	Data         string `json:"data"`
	Timestamp    string `json:"timestamp"`
}

// RuntimeLogsLimit is the maximum number of log entries returned per page.
var RuntimeLogsLimit = 100

func handlerDeploymentRuntimeLogsGet(req *RequestContext) *shttp.Response {
	deploymentID := utils.StringToID(req.Vars()["id"])

	if deploymentID == 0 {
		return shttp.NotFound()
	}

	qs := req.Query()
	sort := strings.ToLower(qs.Get("sort"))

	if sort != "" && sort != "asc" && sort != "desc" {
		return shttp.BadRequest(map[string]any{
			"errors": []string{"sort must be either 'asc' or 'desc'."},
		})
	}

	// The log query is scoped by app, so verify the deployment belongs to the
	// authorized environment before reading anything.
	depl, err := deploy.NewStore().MyDeployment(req.Context(), &deploy.DeploymentsQueryFilters{
		DeploymentID: deploymentID,
		EnvID:        req.Env.ID,
	})

	if err != nil {
		return shttp.Error(err)
	}

	if depl == nil {
		return shttp.NotFound()
	}

	logs, err := applog.NewStore().Logs(req.Context(), &applog.LogQuery{
		AppID:        req.App.ID,
		DeploymentID: deploymentID,
		AfterID:      utils.StringToID(qs.Get("afterId")),
		BeforeID:     utils.StringToID(qs.Get("beforeId")),
		Sort:         sort,
		Limit:        RuntimeLogsLimit,
	})

	if err != nil {
		return shttp.Error(err)
	}

	hasNextPage := len(logs) > RuntimeLogsLimit

	if hasNextPage {
		logs = logs[:RuntimeLogsLimit]
	}

	processed := make([]RuntimeLog, 0, len(logs))

	for _, log := range logs {
		processed = append(processed, RuntimeLog{
			ID:           log.ID.String(),
			AppID:        log.AppID.String(),
			DeploymentID: log.DeploymentID.String(),
			Data:         log.Data,
			Timestamp:    strconv.FormatInt(log.Timestamp, 10),
		})
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"logs":        processed,
			"hasNextPage": hasNextPage,
		},
	}
}
