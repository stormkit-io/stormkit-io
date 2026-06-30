package accesslog

import (
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
