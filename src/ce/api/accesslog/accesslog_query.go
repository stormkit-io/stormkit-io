package accesslog

import (
	"net/url"
	"strings"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// DefaultWindow bounds queries that omit from/to, so a request always prunes to
// a small number of day-partitions instead of scanning the whole retention
// window.
const DefaultWindow = 24 * time.Hour

// SelectLogsParamsFromQuery maps URL query parameters onto select filters,
// defaulting the lower time bound to DefaultWindow ago when from is absent and
// the upper bound to now when to is absent. Both bounds are always set: an open
// upper bound forces the planner to include the create-ahead partitions and
// request_logs_default, neither of which can be pruned away.
// Callers that must restrict the result to a single app or environment are
// expected to overwrite AppID/EnvID after calling this.
func SelectLogsParamsFromQuery(q url.Values) SelectLogsParams {
	from := unixFromQuery(q.Get("from"))
	to := unixFromQuery(q.Get("to"))
	beforeTS, beforeID := decodeCursor(q.Get("cursor"))

	if !from.Valid {
		from = utils.UnixFrom(time.Now().Add(-DefaultWindow))
	}

	if !to.Valid {
		to = utils.UnixFrom(time.Now())
	}

	return SelectLogsParams{
		AppID:         utils.StringToID(q.Get("appId")),
		EnvID:         utils.StringToID(q.Get("envId")),
		DomainID:      utils.StringToID(q.Get("domainId")),
		MinDurationMS: utils.StringToInt(q.Get("minDurationMs")),
		HostName:      q.Get("hostName"),
		ClientIP:      q.Get("clientIp"),
		Method:        q.Get("method"),
		Path:          q.Get("path"),
		Status:        utils.StringToInt(q.Get("status")),
		IsBot:         boolFromQuery(q.Get("isBot")),
		From:          from,
		To:            to,
		BeforeID:      beforeID,
		BeforeTS:      beforeTS,
		Limit:         DefaultLimit,
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

// decodeCursor parses a cursor produced by AccessLog.Cursor into the timestamp
// and id halves of the sort key. A malformed cursor decodes to an unset
// position, which reads as "start from the newest entry" — the same as omitting
// it — rather than failing the request.
func decodeCursor(v string) (utils.Unix, types.ID) {
	raw, err := utils.DecodeString(v)

	if err != nil {
		return utils.Unix{}, 0
	}

	micros, id, found := strings.Cut(string(raw), ".")

	if !found {
		return utils.Unix{}, 0
	}

	parsed := utils.StringToInt64(micros)

	if parsed <= 0 {
		return utils.Unix{}, 0
	}

	return utils.UnixFrom(time.UnixMicro(parsed)), utils.StringToID(id)
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
