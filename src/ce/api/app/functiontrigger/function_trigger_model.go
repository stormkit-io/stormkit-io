package functiontrigger

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"maps"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type Options struct {
	Method  string        `json:"method"`
	URL     string        `json:"url"`
	Payload []byte        `json:"payload,omitempty"`
	Headers shttp.Headers `json:"headers,omitempty"`
}

func (a Options) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *Options) Scan(value any) error {
	if value == nil {
		return nil
	}

	b, ok := value.([]byte)

	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}

type FunctionTrigger struct {
	ID    types.ID `json:"id,string"`
	EnvID types.ID `json:"envId,string"`
	Cron  string   `json:"cron"`
	// Documentation is free-form markdown describing what the trigger is for.
	// Rendered by the UI, never interpreted when the trigger runs.
	Documentation string     `json:"documentation,omitempty"`
	Status        bool       `json:"status"`
	Options       Options    `json:"options,omitempty"`
	NextRunAt     utils.Unix `json:"nextRunAt,omitempty"`
	CreatedAt     utils.Unix `json:"-"`
	UpdatedAt     utils.Unix `json:"-"`
}

// MaskHeaders returns a copy of headers with every VALUE blanked (keys kept).
// Trigger headers routinely carry secrets (Authorization, API keys), so this is
// the single masking rule applied wherever a trigger or its logs are serialized
// to a client — values are never exposed in plaintext outside the stored record
// itself. nil in, nil out.
func MaskHeaders(h shttp.Headers) shttp.Headers {
	if h == nil {
		return nil
	}

	masked := make(shttp.Headers, len(h))

	for k := range h {
		masked[k] = ""
	}

	return masked
}

// ToMap returns the client-ready representation of a trigger with header values
// masked. Internal consumers that need the real headers read Options.Headers
// directly and never go through here.
func (t *FunctionTrigger) ToMap() map[string]any {
	return map[string]any{
		"id":            t.ID.String(),
		"envId":         t.EnvID.String(),
		"cron":          t.Cron,
		"documentation": t.Documentation,
		"status":        t.Status,
		"nextRunAt":     t.NextRunAt.Unix(),
		"options": map[string]any{
			"url":     t.Options.URL,
			"headers": MaskHeaders(t.Options.Headers),
			"method":  t.Options.Method,
			"payload": string(t.Options.Payload),
		},
	}
}

type TriggerLog struct {
	ID        types.ID   `json:"id,string"`
	TriggerID types.ID   `json:"triggerId,string"`
	Request   utils.Map  `json:"request"`
	Response  utils.Map  `json:"response"`
	CreatedAt utils.Unix `json:"createdAt"`
}

// Masked returns a copy of the log with the request header values blanked so a
// trigger's secret headers are never returned to a client. The request map is
// copied shallowly except for the rebuilt headers entry, leaving the original
// (and the stored row) untouched.
func (l TriggerLog) Masked() TriggerLog {
	if l.Request == nil {
		return l
	}

	headers, ok := l.Request["headers"]

	if !ok {
		return l
	}

	request := make(utils.Map, len(l.Request))
	maps.Copy(request, l.Request)

	request["headers"] = maskHeaderValues(headers)
	l.Request = request

	return l
}

// maskHeaderValues blanks every value of a headers map regardless of the
// concrete type it carries — shttp.Headers when freshly built in-process, or
// map[string]any after a JSON round-trip out of the database.
func maskHeaderValues(headers any) any {
	switch h := headers.(type) {
	case shttp.Headers:
		return MaskHeaders(h)
	case map[string]string:
		return MaskHeaders(shttp.Headers(h))
	case map[string]any:
		masked := make(map[string]any, len(h))

		for k := range h {
			masked[k] = ""
		}

		return masked
	default:
		return headers
	}
}
