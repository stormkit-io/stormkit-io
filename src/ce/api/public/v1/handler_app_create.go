package publicapiv1

import (
	"fmt"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type appCreatePost struct {
	Repo        string `json:"repo"`     // owner/slug
	Provider    string `json:"provider"` // github|gitlab|bitbucket
	DisplayName string `json:"displayName,omitempty"`
}

// handlerAppCreate creates a new application linked to a provider repository.
// Authentication requires a team-scoped API key; the app is inserted into that team.
func handlerAppCreate(req *RequestContext) *shttp.Response {
	data := &appCreatePost{}

	if err := req.Post(data); err != nil {
		return shttp.Error(err)
	}

	userID := req.Token.UserID

	if userID == 0 {
		usr, err := user.NewStore().TeamOwner(req.Context(), req.TeamID)

		if err != nil {
			return shttp.Error(err)
		}

		if usr == nil {
			return shttp.Forbidden()
		}

		userID = usr.ID
	}

	// Set these variables and validate them before inserting the app
	myApp := app.New(userID)
	myApp.TeamID = req.TeamID

	if data.Repo != "" {
		myApp.Repo = fmt.Sprintf("%s/%s", strings.ToLower(data.Provider), data.Repo)
	}

	if data.DisplayName != "" {
		myApp.DisplayName = strings.TrimSpace(data.DisplayName)
	}

	if errs := app.Validate(myApp); len(errs) > 0 {
		return shttp.BadRequest(map[string]any{"errors": errs})
	}

	if _, err := app.NewStore().InsertApp(req.Context(), myApp); err != nil {
		return shttp.Error(err)
	}

	if req.License().IsEnterprise() {
		err := audit.FromRequestContext(req).
			WithAction(audit.CreateAction, audit.TypeApp).
			WithDiff(&audit.Diff{
				New: audit.DiffFields{
					AppName: myApp.DisplayName,
					AppRepo: myApp.Repo,
				},
			}).
			WithTeamID(myApp.TeamID).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return app.NewResponse(&app.MyApp{App: myApp})
}
