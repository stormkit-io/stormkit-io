package snippetshandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type SnippetRequest struct {
	Snippets []*buildconf.Snippet `json:"snippets"`
}

func HandlerSnippetsAdd(req *app.RequestContext) *shttp.Response {
	store := buildconf.SnippetsStore()
	data := SnippetRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	if len(data.Snippets) == 0 {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "Nothing to add.",
			},
		}
	}

	snippets := []*buildconf.Snippet{}
	titles := []string{}

	for _, snippet := range data.Snippets {
		if err := validateSnippet(snippet, snippet.Location); err != nil {
			return badRequest(err)
		}

		normalizeRules(snippet.Rules)

		snippet.EnvID = req.EnvID
		snippet.AppID = req.App.ID
		snippets = append(snippets, snippet)
		titles = append(titles, snippet.Title)
	}

	if err := validateDomains(snippets, req.EnvID); err != nil {
		return badRequest(err)
	}

	if err := store.Insert(req.Context(), snippets); err != nil {
		if database.IsDuplicate(err) {
			return duplicateSnippetError()
		}

		return shttp.Error(err)
	}

	if req.License().IsEnterprise() {
		diff := &audit.Diff{
			New: audit.DiffFields{
				Snippets: titles,
			},
		}

		err := audit.FromRequestContext(req).
			WithAction(audit.CreateAction, audit.TypeSnippet).
			WithDiff(diff).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	reset := CalculateResetDomains(req.App.DisplayName, snippets)

	if reset == nil || len(reset) > 0 {
		if err := appcache.Service().Reset(req.EnvID, reset...); err != nil {
			return shttp.Error(err)
		}
	}

	return &shttp.Response{
		Status: http.StatusCreated,
		Data: map[string]any{
			"snippets": toJSON(snippets),
		},
	}
}
