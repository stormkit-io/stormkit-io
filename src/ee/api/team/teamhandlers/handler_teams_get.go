package teamhandlers

import (
	"fmt"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerTeamsGet(req *user.RequestContext) *shttp.Response {
	teams, err := team.NewStore().Teams(req.Context(), req.User.ID)

	if err != nil {
		return shttp.Error(err)
	}

	if teams == nil {
		return shttp.BadRequest()
	}

	data := []map[string]any{}
	slugs := map[string]bool{}
	hasUniqueSlugs := true
	isEnterprise := req.License().IsEnterprise()

	for _, team := range teams {
		// Only show default teams in CE
		if !isEnterprise && !team.IsDefault {
			continue
		}

		if slugs[team.Slug] {
			hasUniqueSlugs = false
		}

		slugs[team.Slug] = true
		data = append(data, team.ToMap())
	}

	if !hasUniqueSlugs {
		for k := range data {
			data[k]["slug"] = fmt.Sprintf("%s-%s", data[k]["slug"], data[k]["id"])
		}
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   data,
	}
}
