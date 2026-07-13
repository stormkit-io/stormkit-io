package apphandlers

import (
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"
)

// UpdateRequest represents an update request.
type UpdateRequest struct {
	// Repo contains the application's repository. It should be in
	// <provider>/<user>/<project> format. Currently, only Bitbucket
	// and Github are supported.
	Repo string `json:"repo"`

	// DisplayName is the application's display name. It is used
	// to prefix the deployment endpoints. It's unique accross the
	// Stormkit application ecosystem.
	DisplayName string `json:"displayName"`
}

// handlerAppUpdate handles updating an app.
func handlerAppUpdate(req *app.RequestContext) *shttp.Response {
	data := &UpdateRequest{}

	if err := req.Post(data); err != nil {
		return shttp.BadRequest().SetError(err)
	}

	if req.App.Repo != "" && data.Repo == "" {
		return shttp.BadRequest(map[string]any{
			"error": "It's not possible to convert an existing app to a bare app. Please create a new app instead.",
		})
	}

	if data.Repo != "" && req.App.Repo == "" {
		return shttp.BadRequest(map[string]any{
			"error": "It's not possible to convert a bare app to a repository app. Please create a new app instead.",
		})
	}

	repo := req.App.Repo
	displayName := req.App.DisplayName

	req.App.Repo = strings.TrimSpace(data.Repo)
	req.App.DisplayName = strings.TrimSpace(data.DisplayName)

	if err := req.App.Validate(); err != nil {
		return shttp.ValidationError(err)
	}

	if err := app.NewStore().UpdateApp(req.Context(), req.App); err != nil {
		// The display name is already in use, try another one.
		if strings.Contains(err.Error(), "duplicate") {
			return shttp.ValidationError(&shttperr.ValidationError{
				Errors: map[string]string{
					"displayName": app.ErrDuplicateDisplayName.Error(),
				},
			})
		}

		return shttp.Error(err)
	}

	if req.License().IsEnterprise() {
		diff := &audit.Diff{
			Old: audit.DiffFields{
				AppName: displayName,
				AppRepo: repo,
			},
			New: audit.DiffFields{
				AppName: req.App.DisplayName,
				AppRepo: req.App.Repo,
			},
		}

		err := audit.FromRequestContext(req).
			WithAction(audit.UpdateAction, audit.TypeApp).
			WithDiff(diff).
			WithAppID(req.App.ID).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	if displayName != req.App.DisplayName {
		reset := []string{
			appcache.DevDomainCacheKey(displayName),
			appcache.DevDomainCacheKey(req.App.DisplayName),
		}

		if err := appcache.Service().Reset(0, reset...); err != nil {
			return shttp.Error(err)
		}
	}

	return shttp.OK()
}
