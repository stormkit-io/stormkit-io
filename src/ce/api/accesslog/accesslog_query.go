package accesslog

import (
	"net/url"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// DefaultWindow bounds queries that omit from/to, so a request always prunes to
// a small number of day-partitions instead of scanning the whole retention
// window.
const DefaultWindow = 24 * time.Hour

// SelectLogsParamsFromQuery maps URL query parameters onto select filters,
// defaulting the lower time bound to DefaultWindow ago when from is absent.
// Callers that must restrict the result to a single app or environment are
// expected to overwrite AppID/EnvID after calling this.
func SelectLogsParamsFromQuery(q url.Values) SelectLogsParams {
	from := unixFromQuery(q.Get("from"))

	if !from.Valid {
		from = utils.UnixFrom(time.Now().Add(-DefaultWindow))
	}

	return SelectLogsParams{
		AppID:    utils.StringToID(q.Get("appId")),
		EnvID:    utils.StringToID(q.Get("envId")),
		DomainID: utils.StringToID(q.Get("domainId")),
		HostName: q.Get("hostName"),
		ClientIP: q.Get("clientIp"),
		Method:   q.Get("method"),
		Path:     q.Get("path"),
		Status:   utils.StringToInt(q.Get("status")),
		IsBot:    boolFromQuery(q.Get("isBot")),
		From:     from,
		To:       unixFromQuery(q.Get("to")),
		BeforeID: utils.StringToID(q.Get("beforeId")),
		Limit:    DefaultLimit,
	}
}

// unixFromQuery parses a query value as unix seconds, returning an invalid Unix
// when absent or unparseable.
func unixFromQuery(v string) utils.Unix {
	secs := utils.StringToInt64(v)

	if secs <= 0 {
		return utils.Unix{}
	}

	return utils.UnixFrom(time.Unix(secs, 0))
}

func boolFromQuery(v string) *bool {
	switch v {
	case "true":
		return utils.Ptr(true)
	case "false":
		return utils.Ptr(false)
	default:
		return nil
	}
}
