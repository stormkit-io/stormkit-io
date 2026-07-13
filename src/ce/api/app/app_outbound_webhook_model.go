package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	null "gopkg.in/guregu/null.v3"
)

const TriggerOnPublish = "on_publish"
const TriggerOnDeploySuccess = "on_deploy_success"
const TriggerOnDeployFailed = "on_deploy_failed"
const TriggerOnCachePurge = "on_cache_purge"

type OutboundWebhook struct {
	WebhookID      types.ID          `json:"id,string"`
	RequestURL     string            `json:"requestUrl"`
	RequestMethod  string            `json:"requestMethod"`
	RequestPayload null.String       `json:"requestPayload"`
	RequestHeaders map[string]string `json:"requestHeaders"`
	TriggerWhen    string            `json:"triggerWhen"`
}

type DispatchOutput struct {
	Error  string `json:"error"`
	Result struct {
		Status int    `json:"status"`
		Body   string `json:"body"`
	} `json:"result"`
}

type OutboundWebhookSettings struct {
	AppID                  types.ID
	DeploymentID           types.ID
	EnvironmentName        string
	DeploymentError        string
	DeploymentEndpoint     string
	DeploymentLogsEndpoint string
	DeploymentStatus       string // success | failed
}

func (wh OutboundWebhook) TriggerOnDeploySuccess() bool {
	return wh.TriggerWhen == TriggerOnDeploySuccess
}

func (wh OutboundWebhook) TriggerOnDeployFailed() bool {
	return wh.TriggerWhen == TriggerOnDeployFailed
}

func (wh OutboundWebhook) TriggerOnPublish() bool {
	return wh.TriggerWhen == TriggerOnPublish
}

func (wh OutboundWebhook) TriggerOnCachePurge() bool {
	return wh.TriggerWhen == TriggerOnCachePurge
}

// Dispatch an outbound webhook
func (wh OutboundWebhook) Dispatch(settings OutboundWebhookSettings) DispatchOutput {
	req := shttp.NewRequestV2(wh.RequestMethod, wh.RequestURL)
	req.Headers(shttp.HeadersFromMap(wh.RequestHeaders))

	if wh.RequestPayload.ValueOrZero() != "" {
		payload := wh.RequestPayload.ValueOrZero()
		patterns := map[string]string{
			"$SK_NOW":      time.Now().Format(time.RFC3339),
			"$SK_NOW_UNIX": fmt.Sprintf("%d", time.Now().Unix()),
		}

		if appID := settings.AppID.String(); appID != "" {
			patterns["$SK_APP_ID"] = appID
		}

		if did := settings.DeploymentID.String(); did != "" {
			patterns["$SK_DEPLOYMENT_ID"] = did
		}

		if ep := settings.DeploymentEndpoint; ep != "" {
			patterns["$SK_DEPLOYMENT_ENDPOINT"] = ep
		}

		if ep := settings.DeploymentLogsEndpoint; ep != "" {
			patterns["$SK_DEPLOYMENT_LOGS_ENDPOINT"] = ep
		}

		if status := settings.DeploymentStatus; status != "" {
			patterns["$SK_DEPLOYMENT_STATUS"] = status
		}

		if err := settings.DeploymentError; err != "" {
			patterns["$SK_DEPLOYMENT_ERROR"] = err
		}

		if env := settings.EnvironmentName; env != "" {
			patterns["$SK_ENVIRONMENT"] = env
		}

		for key, value := range patterns {
			payload = strings.Replace(payload, key, value, -1)
		}

		req.Payload(payload)
	}

	res, err := req.Do()
	data := DispatchOutput{}

	if err != nil {
		data.Error = err.Error()
	}

	if res != nil {
		data.Result.Body = res.String()
		data.Result.Status = res.StatusCode
	}

	return data
}
