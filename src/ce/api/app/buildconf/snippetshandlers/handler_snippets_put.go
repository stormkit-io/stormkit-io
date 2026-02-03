package snippetshandlers

import (
	"errors"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type UpdateRequest struct {
	Snippet *buildconf.Snippet `json:"snippet"`
}

func HandlerSnippetsPut(req *app.RequestContext) *shttp.Response {
	data := UpdateRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	if data.Snippet != nil && data.Snippet.ID == 0 {
		data.Snippet = nil
	}

	if data.Snippet == nil {
		return badRequest(errors.New("no-item"))
	}

	data.Snippet.EnvID = req.EnvID

	if err := validateSnippet(data.Snippet, data.Snippet.Location); err != nil {
		return badRequest(err)
	}

	normalizeRules(data.Snippet.Rules)

	if err := validateDomains([]*buildconf.Snippet{data.Snippet}, req.EnvID); err != nil {
		return badRequest(err)
	}

	store := buildconf.SnippetsStore()
	existingSnippet, err := store.SnippetByID(req.Context(), data.Snippet.ID)

	if err != nil {
		return shttp.Error(err)
	}

	if existingSnippet == nil || existingSnippet.EnvID != req.EnvID {
		return shttp.NotFound()
	}

	diff := &audit.Diff{
		Old: audit.DiffFields{
			SnippetTitle:    existingSnippet.Title,
			SnippetContent:  existingSnippet.Content,
			SnippetLocation: existingSnippet.Location,
			SnippetRules:    existingSnippet.Rules,
			SnippetPrepend:  audit.Bool(existingSnippet.Prepend),
			SnippetEnabled:  audit.Bool(existingSnippet.Enabled),
		},
		New: audit.DiffFields{
			SnippetTitle:    data.Snippet.Title,
			SnippetContent:  data.Snippet.Content,
			SnippetLocation: data.Snippet.Location,
			SnippetRules:    data.Snippet.Rules,
			SnippetPrepend:  audit.Bool(data.Snippet.Prepend),
			SnippetEnabled:  audit.Bool(data.Snippet.Enabled),
		},
	}

	if !diff.HasChanged() {
		return shttp.OK()
	}

	if err := buildconf.SnippetsStore().Update(req.Context(), data.Snippet); err != nil {
		if database.IsDuplicate(err) {
			return duplicateSnippetError()
		}

		return shttp.Error(err)
	}

	if req.License().IsEnterprise() {
		err = audit.FromRequestContext(req).
			WithAction(audit.UpdateAction, audit.TypeSnippet).
			WithDiff(diff).
			WithEnvID(req.EnvID).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	reset := CalculateResetDomains(
		req.App.DisplayName,
		[]*buildconf.Snippet{data.Snippet, existingSnippet},
	)

	if reset == nil || len(reset) > 0 {
		if err := appcache.Service().Reset(req.EnvID, reset...); err != nil {
			return shttp.Error(err)
		}
	}

	return shttp.OK()
}

func duplicateSnippetError() *shttp.Response {
	return &shttp.Response{
		Status: http.StatusConflict,
		Data: map[string]string{
			"error": "A snippet with the same content already exists for this environment.",
		},
	}
}
