package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/accesslog"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// AccessLogsLimit is the maximum number of access log entries returned per page.
var AccessLogsLimit = accesslog.DefaultLimit

func handlerAccessLogsGet(req *RequestContext) *shttp.Response {
	params := accesslog.SelectLogsParamsFromQuery(req.Query())

	// The API key authorizes a single environment, so the app and environment
	// filters are taken from the token rather than from the query string.
	params.AppID = req.App.ID
	params.EnvID = req.Env.ID
	params.Limit = AccessLogsLimit

	logs, err := accesslog.NewStore().SelectLogs(req.Context(), params)

	if err != nil {
		return shttp.Error(err)
	}

	pagination := map[string]any{"hasNextPage": false}
	hasNextPage := len(logs) > AccessLogsLimit

	if hasNextPage {
		logs = logs[:AccessLogsLimit]
		last := logs[len(logs)-1]

		pagination["hasNextPage"] = true
		pagination["cursor"] = last.Cursor()
	}

	items := make([]map[string]any, 0, len(logs))

	for _, l := range logs {
		items = append(items, l.ToMap())
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"accessLogs": items,
			"pagination": pagination,
		},
	}
}
