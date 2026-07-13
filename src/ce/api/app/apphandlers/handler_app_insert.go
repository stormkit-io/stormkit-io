package apphandlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/bitbucket"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/github"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/gitlab"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/model"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type appInsertPost struct {
	model.Model

	Repo     string   `json:"repo"`     // :owner/:slug
	Provider string   `json:"provider"` // github|gitlab|bitbucket
	TeamID   types.ID `json:"teamId,string"`
}

// Validate impleents model.Validate interface.
func (a *appInsertPost) Validate() *shttperr.ValidationError {
	err := &shttperr.ValidationError{}

	if a.Repo != "" {
		pieces := strings.Split(a.Repo, "/")

		if len(pieces) < 2 {
			err.SetError("repo", app.ErrRepoInvalidFormat.Error())
		}

		providers := []string{
			github.ProviderName,
			gitlab.ProviderName,
			bitbucket.ProviderName,
		}

		if !utils.InSliceString(providers, a.Provider) {
			err.SetError("provider", "The provider can only be github, gitlab or bitbucket.")
		}
	}

	return err.ToError()
}

// handlerAppInsert handle app insertion process.
func handlerAppInsert(req *user.RequestContext) *shttp.Response {
	myApp := app.New(req.User.ID)
	data := &appInsertPost{}

	// This already validates
	if err := req.Post(data); err != nil {
		return shttp.Error(err)
	}

	if data.Repo != "" {
		myApp.Repo = fmt.Sprintf("%s/%s", data.Provider, data.Repo)
	}

	myApp.TeamID = data.TeamID

	// Non-enterprise users cannot set TeamID
	if myApp.TeamID == 0 || !req.License().IsEnterprise() {
		var err error
		myApp.TeamID, err = team.NewStore().DefaultTeamID(req.Context(), req.User.ID)

		if err != nil {
			return shttp.Error(err)
		}
	} else {
		t, err := team.NewStore().Team(req.Context(), myApp.TeamID, req.User.ID)

		if err != nil {
			return shttp.Error(err)
		}

		if t == nil || !team.HasWriteAccess(t.CurrentUserRole) {
			return &shttp.Response{
				Status: http.StatusForbidden,
				Data: map[string]string{
					"error": "Team does not exist or user has no access.",
				},
			}
		}
	}

	if _, err := app.NewStore().InsertApp(req.Context(), myApp); err != nil {
		return shttp.Error(err)
	}

	if req.License().IsEnterprise() {
		diff := &audit.Diff{
			New: audit.DiffFields{
				AppName: myApp.DisplayName,
				AppRepo: myApp.Repo,
			},
		}

		err := audit.FromRequestContext(req).
			WithAction(audit.CreateAction, audit.TypeApp).
			WithDiff(diff).
			WithTeamID(myApp.TeamID).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return app.NewResponse(&app.MyApp{App: myApp})
}
