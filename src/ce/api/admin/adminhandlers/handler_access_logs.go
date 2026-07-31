package adminhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/accesslog"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerAccessLogs(req *user.RequestContext) *shttp.Response {
	params := accesslog.SelectLogsParamsFromQuery(req.Query())

	logs, err := accesslog.NewStore().SelectLogs(req.Context(), params)

	if err != nil {
		return shttp.Error(err)
	}

	pagination := map[string]any{"hasNextPage": false}

	if len(logs) > accesslog.DefaultLimit {
		logs = logs[:accesslog.DefaultLimit]
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
