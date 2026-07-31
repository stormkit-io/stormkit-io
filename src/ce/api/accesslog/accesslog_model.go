package accesslog

import (
	"strconv"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"gopkg.in/guregu/null.v3"
)

// AccessLog is a single raw request-level access log entry. Unlike
// analytics.Record it keeps the unmasked client IP, the full user-agent and
// path, and a bot flag rather than dropping bot traffic — it is an operational
// access log, not visitor analytics.
type AccessLog struct {
	ID           types.ID    `json:"id"`
	AppID        types.ID    `json:"appId"`
	EnvID        types.ID    `json:"envId"`
	DeploymentID types.ID    `json:"deploymentId"`
	DomainID     types.ID    `json:"domainId"`
	HostName     string      `json:"hostName"`
	RequestTS    utils.Unix  `json:"requestTimestamp"`
	Method       string      `json:"method"`
	RequestPath  string      `json:"path"`
	StatusCode   int         `json:"statusCode"`
	ClientIP     string      `json:"clientIp"`
	UserAgent    string      `json:"userAgent"`
	Referrer     string      `json:"referrer"`
	IsBot        bool        `json:"isBot"`
	BytesSent    int64       `json:"bytesSent"`
	RequestID    null.String `json:"requestId"`
}

// Cursor encodes the entry's position for keyset pagination.
//
// It carries both halves of the sort key in one opaque value: request_timestamp
// is not unique — concurrent requests land on the same microsecond — and log_id
// alone does not follow timestamp order, so neither identifies a position on its
// own. Encoding them together means a caller cannot pass one without the other.
//
// The timestamp is in microseconds to match request_timestamp's resolution; a
// cursor truncated to whole seconds would skip every entry sharing its second.
func (l AccessLog) Cursor() string {
	if !l.RequestTS.Valid {
		return ""
	}

	micros := strconv.FormatInt(l.RequestTS.Time.UnixMicro(), 10)

	return utils.EncodeToString([]byte(micros + "." + l.ID.String()))
}

func (l AccessLog) ToMap() map[string]any {
	return map[string]any{
		"id":               l.ID.String(),
		"appId":            l.AppID.String(),
		"envId":            l.EnvID.String(),
		"deploymentId":     l.DeploymentID.String(),
		"domainId":         l.DomainID.String(),
		"hostName":         l.HostName,
		"requestTimestamp": l.RequestTS.UnixStr(),
		"method":           l.Method,
		"path":             l.RequestPath,
		"statusCode":       l.StatusCode,
		"clientIp":         l.ClientIP,
		"userAgent":        l.UserAgent,
		"referrer":         l.Referrer,
		"isBot":            l.IsBot,
		"bytesSent":        l.BytesSent,
		"requestId":        l.RequestID.ValueOrZero(),
	}
}
