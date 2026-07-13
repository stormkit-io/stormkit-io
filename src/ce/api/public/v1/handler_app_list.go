package publicapiv1

import (
	"fmt"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// appListLimit is the maximum number of apps returned per page.
const appListLimit = 20

// handlerAppList returns a paginated list of applications that belong to the
// authenticated user. Authentication requires a user-scoped API key.
//
// Query parameters:
//
//	teamId      - Required. ID of the team to scope the query to. The authenticated user must be a member.
//	from        - Offset for pagination (default: 0). Use the value of the next
//	              page's "from" field returned in the response to fetch subsequent pages.
//	repo        - Optional exact (case-insensitive) match on the repository path (e.g. "github/org/repo").
//	displayName - Optional exact (case-insensitive) match on the application display name.
//	filter      - Optional case-insensitive substring match on the application display name or repo.
//				  Ignored if displayName or repo is provided. This is meant for simple client-side search and
//				  therefore it is not documented in the API documentation.
//
// Response:
//
//	apps        - Array of application objects.
//	hasNextPage - Whether more results are available. When true, increment
//	              "from" by the number of items returned to get the next page.
func handlerAppList(req *RequestContext) *shttp.Response {
	q := req.Query()
	v := &Validators{}

	from, err := v.ToInt(q.Get("from"), "from")

	if err != nil {
		return shttp.BadRequest(map[string]any{"errors": []string{err.Error()}})
	}

	repo, repoValid := v.NormalizeRepo(q.Get("repo"))

	if !repoValid {
		return shttp.BadRequest(map[string]any{
			"errors": []string{"The 'repo' parameter must be in the format 'github/org/repo', 'gitlab/org/repo', or 'bitbucket/org/repo'"},
		})
	}

	myApps, err := app.NewStore().Apps(req.Context(), app.AppsArgs{
		Repo:        repo,
		DisplayName: q.Get("displayName"),
		Filter:      q.Get("filter"),
		TeamID:      req.TeamID,
		From:        from,
		Limit:       appListLimit + 1,
	})

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to fetch apps for team id %d: %s", req.TeamID, err.Error()))
	}

	hasNextPage := false

	if len(myApps) > appListLimit {
		hasNextPage = true
		myApps = myApps[:appListLimit]
	}

	apps := make([]map[string]any, 0, len(myApps))

	for _, a := range myApps {
		apps = append(apps, a.JSON())
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"apps":        apps,
			"hasNextPage": hasNextPage,
		},
	}
}
