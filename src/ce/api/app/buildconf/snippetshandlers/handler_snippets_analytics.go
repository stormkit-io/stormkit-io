package snippetshandlers

import (
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// AnalyticsSnippetTitle is the reserved title that identifies the managed
// client-analytics snippet, so enable/disable can find and update its own row.
const AnalyticsSnippetTitle = "Stormkit Analytics"

// analyticsSnippetSrc is the same-origin path the served script lives at. It is
// also used as a content fallback when matching the managed snippet, so a user
// renaming the title does not orphan it.
const analyticsSnippetSrc = "/_stormkit/analytics.js"

// analyticsSnippetContent is the canonical managed snippet. It loads the
// same-origin script and carries the per-request id token, interpolated at inject
// time (the snippet is created with Interpolate=true).
const analyticsSnippetContent = `<script src="` + analyticsSnippetSrc + `" data-sk-rid="{{SK_REQUEST_ID}}" async></script>`

// HandlerAnalyticsSnippetEnable creates (or re-enables) the managed client-side
// analytics snippet for the environment. Enterprise-gated, matching the collect
// endpoint that ingests the beacons this script sends.
func HandlerAnalyticsSnippetEnable(req *app.RequestContext) *shttp.Response {
	if !req.License().IsEnterprise() {
		return shttp.Forbidden()
	}

	store := buildconf.SnippetsStore()
	existing, err := findAnalyticsSnippet(req)

	if err != nil {
		return shttp.Error(err)
	}

	if existing != nil {
		existing.Content = analyticsSnippetContent
		existing.Location = buildconf.SnippetLocationHead
		existing.Enabled = true
		existing.Interpolate = true

		if err := store.Update(req.Context(), existing); err != nil {
			return shttp.Error(err)
		}

		return analyticsSnippetResponse(req, existing, http.StatusOK)
	}

	snippet := &buildconf.Snippet{
		AppID:       req.App.ID,
		EnvID:       req.EnvID,
		Title:       AnalyticsSnippetTitle,
		Content:     analyticsSnippetContent,
		Location:    buildconf.SnippetLocationHead,
		Enabled:     true,
		Interpolate: true,
	}

	if err := store.Insert(req.Context(), []*buildconf.Snippet{snippet}); err != nil {
		return shttp.Error(err)
	}

	return analyticsSnippetResponse(req, snippet, http.StatusCreated)
}

// HandlerAnalyticsSnippetDisable removes the managed analytics snippet. Idempotent
// (returns 200 when none exists) and not enterprise-gated, so a lapsed license can
// still turn it off.
func HandlerAnalyticsSnippetDisable(req *app.RequestContext) *shttp.Response {
	existing, err := findAnalyticsSnippet(req)

	if err != nil {
		return shttp.Error(err)
	}

	if existing == nil {
		return shttp.OK()
	}

	if err := buildconf.SnippetsStore().Delete(req.Context(), []types.ID{existing.ID}, req.EnvID); err != nil {
		return shttp.Error(err)
	}

	if err := appcache.Service().Reset(req.EnvID); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}

// findAnalyticsSnippet locates the managed snippet by its reserved title, falling
// back to a content match on the served script path so a title edit does not
// orphan the row.
func findAnalyticsSnippet(req *app.RequestContext) (*buildconf.Snippet, error) {
	snippets, err := buildconf.SnippetsStore().SnippetsByEnvID(req.Context(), buildconf.SnippetFilters{
		EnvID: req.EnvID,
	})

	if err != nil {
		return nil, err
	}

	for _, snippet := range snippets {
		if snippet.Title == AnalyticsSnippetTitle || strings.Contains(snippet.Content, analyticsSnippetSrc) {
			return snippet, nil
		}
	}

	return nil, nil
}

func analyticsSnippetResponse(req *app.RequestContext, snippet *buildconf.Snippet, status int) *shttp.Response {
	if err := appcache.Service().Reset(req.EnvID); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: status,
		Data:   map[string]any{"snippet": snippet.JSON()},
	}
}
