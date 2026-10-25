package hosting

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
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
	// Search is the landing URL's query string, sent only for landings the
	// server never saw, so their campaign is not lost.
	Search string `json:"search"`
}

// collectEngagement is the foreground time the script measured on a page
// since its last report.
type collectEngagement struct {
	Path    string `json:"path"`
	ViewID  string `json:"viewId"`
	Seconds int    `json:"seconds"`
}

// maxViewIDLength bounds the client-chosen page view id.
const maxViewIDLength = 64

// maxEventNameLength bounds a custom event name, in runes. Longer names are
// truncated, like campaign values.
const maxEventNameLength = 200

// maxBeaconPathLength bounds a path the script reports, in bytes. Real paths
// are far shorter, so an item with a longer one is dropped.
const maxBeaconPathLength = 2048

// maxMetadataLength bounds the raw metadata of one custom event, in bytes.
// Larger metadata is dropped while the event itself is kept.
const maxMetadataLength = 4 * 1024

type collectPayload struct {
	Events      []collectEvent      `json:"events"`
	Pageviews   []collectPageview   `json:"pageviews"`
	Engagements []collectEngagement `json:"engagements"`
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

	// The domain is opted out of visitor analytics, so neither its pageviews nor
	// its custom events are recorded. Answered as success so a cached tracking
	// script on an excluded host does not report errors to the visitor.
	if req.Host.Config.AnalyticsExcluded {
		return &shttp.Response{Status: http.StatusNoContent}
	}

	// The script always posts to its own origin, so a browser beacon from any
	// other origin is a third-party page writing into this site's analytics.
	// Server-to-server calls send neither header and are unaffected.
	if isCrossOriginBeacon(req) {
		return &shttp.Response{Status: http.StatusNoContent}
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

	// Analytics are read per domain, so rows for a host without one (a
	// deployment preview or the env's default URL) could never be shown.
	if cfg.DomainID == 0 && !config.IsDevelopment() {
		return &shttp.Response{Status: http.StatusNoContent}
	}

	visitorID := analytics.VisitorID(analytics.VisitorIDParams{
		IP:        req.RemoteIP(),
		UserAgent: userAgent,
	})

	events := make([]analytics.Event, 0, len(payload.Events))

	for _, e := range payload.Events {
		if e.Name == "" || len(e.Path) > maxBeaconPathLength {
			continue
		}

		path := beaconPath(e.Path)

		events = append(events, analytics.Event{
			AppID:       cfg.AppID,
			EnvID:       cfg.EnvID,
			DomainID:    cfg.DomainID,
			VisitorID:   null.StringFrom(visitorID),
			EventName:   eventName(e.Name),
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
		path := beaconPath(pv.Path)

		if path == "" || len(pv.Path) > maxBeaconPathLength {
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
				Campaign:    analytics.CampaignFromRawQuery(pv.Search),
			},
		})
	}

	engagements := make([]analytics.Engagement, 0, len(payload.Engagements))

	// One page view's reports in a beacon are merged, so repeating a view id
	// cannot multiply the per-report cap.
	byViewID := map[string]int{}

	for _, e := range payload.Engagements {
		path := beaconPath(e.Path)
		viewID := stripNullChars(e.ViewID)

		if path == "" || len(e.Path) > maxBeaconPathLength || viewID == "" || len(viewID) > maxViewIDLength || e.Seconds <= 0 {
			continue
		}

		seconds := min(e.Seconds, analytics.MaxEngagementSeconds)

		if i, ok := byViewID[viewID]; ok {
			engagements[i].Seconds = min(engagements[i].Seconds+seconds, analytics.MaxEngagementSeconds)
			continue
		}

		byViewID[viewID] = len(engagements)

		engagements = append(engagements, analytics.Engagement{
			DomainID:    cfg.DomainID,
			RequestPath: path,
			ViewID:      viewID,
			Seconds:     seconds,
			EngagedTS:   utils.NewUnix(),
		})
	}

	if len(engagements) > 0 {
		Queue(&jobs.HostingRecord{
			AppID:         cfg.AppID,
			EnvID:         cfg.EnvID,
			DeploymentID:  cfg.DeploymentID,
			HostName:      req.Host.Name,
			BillingUserID: cfg.BillingUserID,
			Engagements:   engagements,
		})
	}

	return &shttp.Response{Status: http.StatusNoContent}
}

// isCrossOriginBeacon reports whether a browser sent the beacon from a page on
// another origin, judged by Sec-Fetch-Site, or by Origin when the browser does
// not send fetch metadata.
func isCrossOriginBeacon(req *RequestContext) bool {
	if site := req.Header.Get("Sec-Fetch-Site"); site != "" {
		return site != "same-origin" && site != "none"
	}

	origin := req.Header.Get("Origin")

	if origin == "" {
		return false
	}

	parsed, err := url.Parse(origin)

	if err != nil || parsed.Hostname() == "" {
		return true
	}

	// Host names can carry a port in local development.
	return !strings.EqualFold(parsed.Hostname(), req.Host.Name) && !strings.EqualFold(parsed.Host, req.Host.Name)
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

	if len(payload.Events) == 0 && len(payload.Pageviews) == 0 && len(payload.Engagements) == 0 {
		return payload, false
	}

	if len(payload.Events) > maxEventsPerBeacon {
		payload.Events = payload.Events[:maxEventsPerBeacon]
	}

	if len(payload.Pageviews) > maxEventsPerBeacon {
		payload.Pageviews = payload.Pageviews[:maxEventsPerBeacon]
	}

	if len(payload.Engagements) > maxEventsPerBeacon {
		payload.Engagements = payload.Engagements[:maxEventsPerBeacon]
	}

	return payload, true
}

// beaconPath returns a path the script sent in the form the server records
// page views under. Browsers report location.pathname percent-encoded
// (/caf%C3%A9) while the server records the decoded path (/café), so it is
// decoded the same way; a malformed encoding is kept as sent.
func beaconPath(p string) string {
	if decoded, err := url.PathUnescape(p); err == nil {
		p = decoded
	}

	return stripNullChars(p)
}

// stripNullChars removes NUL bytes, which Postgres text columns cannot store and
// which would otherwise abort the shared batch insert.
func stripNullChars(s string) string {
	if !strings.ContainsRune(s, 0) {
		return s
	}

	return strings.ReplaceAll(s, "\x00", "")
}

// eventName returns a custom event name without NUL bytes and capped at
// maxEventNameLength runes.
func eventName(name string) string {
	name = stripNullChars(name)

	if utf8.RuneCountInString(name) > maxEventNameLength {
		name = string([]rune(name)[:maxEventNameLength])
	}

	return name
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

// sanitizeMetadata drops client metadata that Postgres jsonb cannot store, since
// a single bad value would abort the shared batch insert. The value is decoded
// and re-encoded first: Go's encoder replaces lone UTF-16 surrogate escapes
// (e.g. \ud800), which jsonb rejects, with U+FFFD. A NUL escape survives the
// round trip and is still valid JSON, so it is checked afterwards. Metadata
// over maxMetadataLength is dropped too.
func sanitizeMetadata(raw json.RawMessage) null.String {
	if len(raw) == 0 || len(raw) > maxMetadataLength {
		return null.String{}
	}

	var value any

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	if err := decoder.Decode(&value); err != nil {
		return null.String{}
	}

	canonical, err := json.Marshal(value)

	if err != nil {
		return null.String{}
	}

	if bytes.IndexByte(canonical, 0) != -1 || bytes.Contains(bytes.ToLower(canonical), jsonNullEscape) {
		return null.String{}
	}

	return null.StringFrom(string(canonical))
}
