package apploghandlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/applog"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type Log struct {
	ID           string `json:"id"`
	AppID        string `json:"appId"`
	DeploymentID string `json:"deploymentId"`
	Data         string `json:"data"`
	Timestamp    string `json:"timestamp"`
}

var LogsLimit = 100

func handlerLogsGet(req *app.RequestContext) *shttp.Response {
	qs := req.Query()

	query := &applog.LogQuery{
		AppID:        req.App.ID,
		DeploymentID: utils.StringToID(qs.Get("deploymentId")),
		AfterID:      utils.StringToID(qs.Get("afterId")),
		BeforeID:     utils.StringToID(qs.Get("beforeId")),
		Sort:         strings.ToLower(qs.Get("sort")),
		Limit:        LogsLimit,
	}

	if query.DeploymentID == 0 {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "deploymentId is required parameter.",
			},
		}
	}

	if query.Sort != "" && query.Sort != "asc" && query.Sort != "desc" {
		return shttp.BadRequest(map[string]any{
			"error": "sort must be either 'asc' or 'desc'.",
		})
	}

	logs, err := applog.NewStore().Logs(req.Context(), query)

	if err != nil {
		return shttp.Error(err)
	}

	processed := []Log{}
	hasNextPage := false

	if len(logs) >= LogsLimit {
		logs = logs[:len(logs)-1]
		hasNextPage = true
	}

	for _, log := range logs {
		processed = append(processed, Log{
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
