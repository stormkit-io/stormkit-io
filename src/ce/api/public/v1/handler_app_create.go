package publicapiv1

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type appCreatePost struct {
	// Repo accepts any of:
	//   - a full URL: https://github.com/owner/repo, file:///abs/path
	//   - Stormkit style: github/owner/repo, local/abs/path
	//   - bare owner/repo (provider must be set, or defaults to github)
	Repo string `json:"repo"`

	// Provider is optional when Repo carries its own scheme/prefix.
	// One of github|gitlab|bitbucket|local.
	Provider    string `json:"provider,omitempty"`
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
		repo := strings.TrimSpace(data.Repo)
		provider := strings.ToLower(strings.TrimSpace(data.Provider))

		// When the repo string carries its own provider hint (URL scheme or
		// known prefix), derive provider from it and ignore an explicit
		// `provider` field. Otherwise fall back to concatenating with the
		// explicit provider — preserving the legacy two-field form.
		if repoHasProviderHint(repo) {
			if p, slug := utils.ParseRepoWithProvider(repo); slug != "" && p != "" {
				provider = p
				repo = slug
			}
		}

		myApp.Repo = fmt.Sprintf("%s/%s", provider, repo)
	}

	if trimmed := strings.TrimSpace(data.DisplayName); trimmed != "" {
		// If a non-empty display name is provided, trim it and use it. Otherwise, it will use the default set by app.New().
		myApp.DisplayName = trimmed
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

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"app": myApp.JSON(),
		},
	}
}

// repoHasProviderHint reports whether the input carries enough information
// to derive a provider on its own (URL scheme or known Stormkit prefix).
func repoHasProviderHint(repo string) bool {
	lower := strings.ToLower(repo)

	for _, prefix := range []string{"http://", "https://", "file://", "github/", "gitlab/", "bitbucket/", "local/"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}

	return false
}
