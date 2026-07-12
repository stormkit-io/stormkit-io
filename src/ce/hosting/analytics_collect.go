package hosting

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/limiter"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"gopkg.in/guregu/null.v3"
)

// reservedPathPrefix namespaces Stormkit's own platform endpoints (auth, the
// analytics beacon, etc.) so they can be routed internally and excluded from
// visitor analytics.
const reservedPathPrefix = "/_stormkit/"

const collectPath = reservedPathPrefix + "collect"

// maxEventsPerBeacon bounds how many events a single beacon request can submit,
// to limit abuse of the unauthenticated collect endpoint.
const maxEventsPerBeacon = 20

// collectLimiter throttles the unauthenticated collect endpoint per visitor IP.
// Beacons are bursty (initial pageview plus events on load, then SPA
// navigations), so the burst is generous while the sustained rate stops a single
// client from flooding the ingestion pipeline. The Cleanup goroutine evicts idle
// IPs so the in-memory map does not grow unbounded.
var collectLimiter = limiter.NewStore(&limiter.Options{
	Limit:    120,
	Burst:    30,
	Duration: time.Minute,
})

func init() {
	go limiter.Cleanup()
}

// jsonNullEscape is the JSON unicode escape for a NUL code point, which is valid
// JSON but cannot be stored in a Postgres jsonb column.
var jsonNullEscape = []byte{'\\', 'u', '0', '0', '0', '0'}

type collectEvent struct {
	Name      string          `json:"name"`
	Path      string          `json:"path"`
	RequestID string          `json:"requestId"`
	Metadata  json.RawMessage `json:"metadata"`
}

type collectPageview struct {
	Path      string `json:"path"`
	Referrer  string `json:"referrer"`
	RequestID string `json:"requestId"`
}

type collectPayload struct {
	Events    []collectEvent    `json:"events"`
	Pageviews []collectPageview `json:"pageviews"`
}

// handleCollect handles the client-side (and server-to-server) analytics beacon
// at /_stormkit/collect. It resolves the app/env/domain from the host, computes
// a cookieless visitor id, drops bots, and queues the events through the same
// async pipeline as server-side analytics. Custom events are enterprise-only,
// matching server-side analytics.
func handleCollect(req *RequestContext) (res *shttp.Response) {
	defer func() { res = finalizeReserved(req, res) }()

	if req.Method != http.MethodPost {
		return &shttp.Response{Status: http.StatusMethodNotAllowed}
	}

	if req.Host == nil || req.Host.Config == nil || !req.Host.Config.IsEnterprise {
		return &shttp.Response{Status: http.StatusNotFound}
	}

	if !collectLimiter.Get(req.RemoteIP()).Limiter.Allow() {
		return &shttp.Response{Status: http.StatusTooManyRequests}
	}

	userAgent := req.UserAgent()

	// Bots never reach human-facing aggregates; drop silently.
	if analytics.IsBot(userAgent) || !analytics.IsUtf8(userAgent) {
		return &shttp.Response{Status: http.StatusNoContent}
	}

	payload, ok := parseCollectBody(req.Body)

	if !ok {
		return &shttp.Response{Status: http.StatusBadRequest}
	}

	cfg := req.Host.Config
	visitorID := analytics.VisitorID(analytics.VisitorIDParams{
		IP:        req.RemoteIP(),
		UserAgent: userAgent,
	})

	events := make([]analytics.Event, 0, len(payload.Events))

	for _, e := range payload.Events {
		if e.Name == "" {
			continue
		}

		path := stripNullChars(e.Path)

		events = append(events, analytics.Event{
			AppID:       cfg.AppID,
			EnvID:       cfg.EnvID,
			DomainID:    cfg.DomainID,
			VisitorID:   null.StringFrom(visitorID),
			EventName:   stripNullChars(e.Name),
			RequestPath: null.NewString(path, path != ""),
			// The request id is only present when Stormkit injects the
			// client-side script; server-side events leave it null.
			RequestID: sanitizeRequestID(e.RequestID),
			EventTS:   utils.NewUnix(),
			Metadata:  sanitizeMetadata(e.Metadata),
		})
	}

	if len(events) > 0 {
		Queue(&jobs.HostingRecord{
			AppID:         cfg.AppID,
			EnvID:         cfg.EnvID,
			DeploymentID:  cfg.DeploymentID,
			HostName:      req.Host.Name,
			BillingUserID: cfg.BillingUserID,
			Events:        events,
		})
	}

	// Client-side (SPA) pageviews are recorded into the analytics table tagged
	// as "client", so they can be told apart from server-recorded hits.
	for _, pv := range payload.Pageviews {
		path := stripNullChars(pv.Path)

		if path == "" {
			continue
		}

		referrer := analytics.NormalizeReferrer(pv.Referrer)

		Queue(&jobs.HostingRecord{
			AppID:         cfg.AppID,
			EnvID:         cfg.EnvID,
			DeploymentID:  cfg.DeploymentID,
			HostName:      req.Host.Name,
			BillingUserID: cfg.BillingUserID,
			Analytics: &analytics.Record{
				AppID:       cfg.AppID,
				EnvID:       cfg.EnvID,
				DomainID:    cfg.DomainID,
				VisitorIP:   req.RemoteIP(),
				RequestPath: path,
				RequestTS:   utils.NewUnix(),
				StatusCode:  http.StatusOK,
				UserAgent:   null.NewString(userAgent, userAgent != ""),
				Referrer:    null.NewString(referrer, referrer != ""),
				RequestID:   sanitizeRequestID(pv.RequestID),
				Source:      null.StringFrom("client"),
			},
		})
	}

	return &shttp.Response{Status: http.StatusNoContent}
}

func parseCollectBody(body io.ReadCloser) (collectPayload, bool) {
	var payload collectPayload

	if body == nil {
		return payload, false
	}

	defer body.Close()

	data, err := io.ReadAll(io.LimitReader(body, 64*1024))

	if err != nil {
		return payload, false
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		return payload, false
	}

	if len(payload.Events) == 0 && len(payload.Pageviews) == 0 {
		return payload, false
	}

	if len(payload.Events) > maxEventsPerBeacon {
		payload.Events = payload.Events[:maxEventsPerBeacon]
	}

	if len(payload.Pageviews) > maxEventsPerBeacon {
		payload.Pageviews = payload.Pageviews[:maxEventsPerBeacon]
	}

	return payload, true
}

// stripNullChars removes NUL bytes, which Postgres text columns cannot store and
// which would otherwise abort the shared batch insert.
func stripNullChars(s string) string {
	if !strings.ContainsRune(s, 0) {
		return s
	}

	return strings.ReplaceAll(s, "\x00", "")
}

// sanitizeRequestID keeps the client-supplied request id only when it is a valid
// UUID, returning an invalid (NULL) value otherwise. The column is typed uuid, so
// a malformed value would abort the batch insert for every tenant in it.
func sanitizeRequestID(id string) null.String {
	if id == "" {
		return null.String{}
	}

	if _, err := uuid.Parse(id); err != nil {
		return null.String{}
	}

	return null.StringFrom(id)
}

// sanitizeMetadata drops client metadata that Postgres jsonb cannot store. The
// value is already well-formed JSON (it parsed as part of the payload), but jsonb
// rejects a NUL escape or a raw NUL byte, either of which would abort the shared
// batch insert; drop the bad value rather than fail.
func sanitizeMetadata(raw json.RawMessage) null.String {
	if len(raw) == 0 {
		return null.String{}
	}

	if bytes.IndexByte(raw, 0) != -1 || bytes.Contains(bytes.ToLower(raw), jsonNullEscape) {
		return null.String{}
	}

	return null.StringFrom(string(raw))
}
