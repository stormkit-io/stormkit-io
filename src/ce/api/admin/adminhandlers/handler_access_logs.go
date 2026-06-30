package adminhandlers

import (
	"net/http"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/accesslog"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// accessLogWindow bounds queries that omit from/to, so a request always prunes
// to a small number of day-partitions instead of scanning the whole retention
// window.
const accessLogWindow = 24 * time.Hour

func handlerAccessLogs(req *user.RequestContext) *shttp.Response {
	query := req.Query()

	from := accessLogUnix(query.Get("from"))
	to := accessLogUnix(query.Get("to"))

	if !from.Valid {
		from = utils.UnixFrom(time.Now().Add(-accessLogWindow))
	}

	params := accesslog.SelectLogsParams{
		AppID:    utils.StringToID(query.Get("appId")),
		DomainID: utils.StringToID(query.Get("domainId")),
		HostName: query.Get("hostName"),
		ClientIP: query.Get("clientIp"),
		Method:   query.Get("method"),
		Path:     query.Get("path"),
		Status:   utils.StringToInt(query.Get("status")),
		IsBot:    accessLogBool(query.Get("isBot")),
		From:     from,
		To:       to,
		BeforeID: utils.StringToID(query.Get("beforeId")),
		Limit:    accesslog.DefaultLimit,
	}

	logs, err := accesslog.NewStore().SelectLogs(req.Context(), params)

	if err != nil {
		return shttp.Error(err)
	}

	pagination := map[string]any{"hasNextPage": false}
	logsLen := len(logs)

	if logsLen > accesslog.DefaultLimit {
		pagination["hasNextPage"] = true
		pagination["beforeId"] = logs[logsLen-2].ID.String()
		logs = logs[:logsLen-1]
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

// accessLogUnix parses a query value as unix seconds, returning an invalid Unix
// when absent or unparseable.
func accessLogUnix(v string) utils.Unix {
	secs := utils.StringToInt64(v)

	if secs <= 0 {
		return utils.Unix{}
	}

	return utils.UnixFrom(time.Unix(secs, 0))
}

func accessLogBool(v string) *bool {
	switch v {
	case "true":
		return utils.Ptr(true)
	case "false":
		return utils.Ptr(false)
	default:
		return nil
	}
}
