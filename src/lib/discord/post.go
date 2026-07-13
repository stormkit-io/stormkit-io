package discord

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

type PayloadField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type PayloadEmbed struct {
	URL       string         `json:"url,omitempty"`
	Title     string         `json:"title"`
	Timestamp string         `json:"timestamp"` // ISO8601 format
	Fields    []PayloadField `json:"fields"`
}

// Payload represents a discord message payload. For more
// information see the following documentation:
// https://discord.com/developers/docs/resources/webhook#edit-webhook-message-jsonform-params
type Payload struct {
	Embeds []PayloadEmbed `json:"embeds"`
}

// Notify sends a message to the specified channel with the given payload.
func Notify(channel string, payload Payload) {
	if channel == "" {
		return
	}

	headers := make(http.Header)
	headers.Add("Content-Type", "application/json")

	res, err := shttp.NewRequestV2(shttp.MethodPost, channel).
		Headers(headers).
		Payload(payload).
		Do()

	if err != nil {
		slog.Errorf("failed while posting a request to Discord: %v", err)
		return
	}

	if res != nil && res.Body != nil {
		defer res.Body.Close()
	}
}
