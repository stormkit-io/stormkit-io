package apphandlers

import (
	"net/http"
	"strconv"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/model"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// AppListLimit represents the maximum number of items that can
// be retrieved at once from the database.
var AppListLimit = 20

type appIndexRequest struct {
	model.Model

	TeamID types.ID
	Filter string
	From   int
	Limit  int
}

// Validate validates the deploy list request.
func (air *appIndexRequest) Validate() *shttperr.ValidationError {
	err := &shttperr.ValidationError{}

	if air.From < 0 {
		err.SetError("from", "From cannot be smaller than 0")
	}

	return err.ToError()
}

// handlerAppIndex returns the list of apps that user has.
func handlerAppIndex(req *user.RequestContext) *shttp.Response {
	q := req.Query()

	air := &appIndexRequest{}
	air.Limit = AppListLimit
	air.TeamID = utils.StringToID(q.Get("teamId"))
	air.From, _ = strconv.Atoi(q.Get("from"))
	air.Filter = q.Get("filter")

	if err := air.Validate(); err != nil {
		return shttp.Error(err)
	}

	teamStore := team.NewStore()

	// If the team id is not provided, fallback to the default team.
	// Otherwise check if the user is a member.
	if air.TeamID == 0 || !req.License().IsEnterprise() {
		teamID, err := teamStore.DefaultTeamID(req.Context(), req.User.ID)

		if err != nil {
			return shttp.Error(err)
		}

		air.TeamID = teamID
	} else if !teamStore.IsMember(req.Context(), req.User.ID, air.TeamID) {
		return shttp.NotAllowed()
	}

	myApps, err := app.NewStore().Apps(req.Context(), app.AppsArgs{
		TeamID: air.TeamID,
		Filter: air.Filter,
		From:   air.From,
		Limit:  air.Limit + 1,
	})

	if err != nil {
		return shttp.Error(err)
	}

	hasNextPage := false

	if len(myApps) > air.Limit {
		hasNextPage = true
		myApps = myApps[:len(myApps)-1]
	}

	apps := []map[string]any{}

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
