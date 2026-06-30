package hosting

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

//go:embed analytics.js
var analyticsScript []byte

const analyticsScriptPath = reservedPathPrefix + "analytics.js"

// embeddedScriptETag is the content hash of the embedded default, served when no
// admin override is configured so clients refetch only when the script changes.
var embeddedScriptETag = etagFor(analyticsScript)

func etagFor(content []byte) string {
	sum := sha256.Sum256(content)
	return `"` + hex.EncodeToString(sum[:16]) + `"`
}

// analyticsScriptContent returns the client tracking script and its ETag. It
// serves an admin-supplied override when configured — the seam that lets a
// deployment pick up a newer script (via the Admin UI) without upgrading Stormkit
// — and falls back to the embedded default. The override is read from the cached
// instance config, so this stays cheap per request.
func analyticsScriptContent(ctx context.Context) ([]byte, string) {
	cfg, err := admin.Store().Config(ctx)

	if err != nil {
		slog.Errorf("error reading admin config for analytics script: %s", err.Error())
		return analyticsScript, embeddedScriptETag
	}

	return resolveAnalyticsScript(cfg)
}

// resolveAnalyticsScript picks the effective script + ETag from the instance
// config: an admin override when set (ETag from its stored Hash, falling back to a
// computed hash), otherwise the embedded default.
func resolveAnalyticsScript(cfg admin.InstanceConfig) ([]byte, string) {
	if cfg.AnalyticsScript != nil && cfg.AnalyticsScript.Content != "" {
		etag := etagFor([]byte(cfg.AnalyticsScript.Content))

		if cfg.AnalyticsScript.Hash != "" {
			etag = `"` + cfg.AnalyticsScript.Hash + `"`
		}

		return []byte(cfg.AnalyticsScript.Content), etag
	}

	return analyticsScript, embeddedScriptETag
}

// WithAnalyticsScript serves the client-side analytics script at a stable URL.
func WithAnalyticsScript(req *RequestContext) (*shttp.Response, error) {
	if req.URL().Path != analyticsScriptPath {
		return nil, nil
	}

	content, etag := analyticsScriptContent(req.Context())

	headers := http.Header{}
	headers.Set("Content-Type", "application/javascript; charset=utf-8")
	headers.Set("Cache-Control", "public, max-age=3600")
	headers.Set("ETag", etag)

	if req.Header.Get("If-None-Match") == etag {
		return &shttp.Response{Status: http.StatusNotModified, Headers: headers}, nil
	}

	return &shttp.Response{
		Status:  http.StatusOK,
		Headers: headers,
		Data:    content,
	}, nil
}
