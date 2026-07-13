package publicapiv1

import (
	"fmt"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerTeamList returns all teams the authenticated user belongs to.
// In CE, only the default team is returned. Authentication requires a
// user-scoped API key.
func handlerTeamList(req *RequestContext) *shttp.Response {
	teams, err := team.NewStore().Teams(req.Context(), req.Token.UserID)

	if err != nil {
		return shttp.Error(err)
	}

	isEnterprise := req.License().IsEnterprise()
	data := []map[string]any{}
	slugs := map[string]bool{}
	hasUniqueSlugs := true

	for _, t := range teams {
		if !isEnterprise && !t.IsDefault {
			continue
		}

		if slugs[t.Slug] {
			hasUniqueSlugs = false
		}

		slugs[t.Slug] = true
		data = append(data, t.ToMap())
	}

	if !hasUniqueSlugs {
		for k := range data {
			data[k]["slug"] = fmt.Sprintf("%s-%s", data[k]["slug"], data[k]["id"])
		}
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   map[string]any{"teams": data},
	}
}
