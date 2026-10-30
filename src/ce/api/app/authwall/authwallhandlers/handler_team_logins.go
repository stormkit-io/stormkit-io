package authwallhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const teamLoginsLimit = 50

// handlerTeamLogins returns a page of the team's Auth Wall logins.
func handlerTeamLogins(req *user.RequestContext) *shttp.Response {
	t, res := teamAccess{req: req, manage: true}.team(utils.StringToID(req.Query().Get("teamId")))

	if res != nil {
		return res
	}

	from := utils.StringToInt(req.Query().Get("from"))

	if from < 0 {
		from = 0
	}

	logins, err := authwall.Store().TeamLogins(req.Context(), authwall.TeamLoginsParams{
		TeamID: t.ID,
		From:   from,
		Limit:  teamLoginsLimit + 1,
	})

	if err != nil {
		return shttp.Error(err)
	}

	hasNextPage := len(logins) > teamLoginsLimit

	if hasNextPage {
		logins = logins[:teamLoginsLimit]
	}

	data := []map[string]any{}

	for i := range logins {
		data = append(data, loginInput{}.json(&logins[i]))
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"logins":             data,
			"hasNextPage":        hasNextPage,
			"magicLinkAvailable": magicLinkAvailable(),
		},
	}
}
