package publicapiv1

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gosimple/slug"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// TeamCreateRequest is the body accepted by handlerTeamCreate.
type TeamCreateRequest struct {
	Name string `json:"name"`
}

// handlerTeamCreate creates a new team owned by the authenticated user.
// Creating additional teams is an enterprise capability — CE instances only
// ever have the default team — so the handler is gated on the license.
// Authentication requires a user-scoped API key.
func handlerTeamCreate(req *RequestContext) *shttp.Response {
	if resp := req.RequireEnterprise(); resp != nil {
		return resp
	}

	data := TeamCreateRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	data.Name = strings.TrimSpace(data.Name)

	if data.Name == "" {
		return shttp.BadRequest(map[string]any{
			"error": "Team name is a required field.",
		})
	}

	store := team.NewStore()
	teams, err := store.Teams(req.Context(), req.Token.UserID)

	if err != nil {
		return shttp.Error(err)
	}

	if len(teams) >= team.MAX_TEAMS_PER_USER {
		return shttp.BadRequest(map[string]any{
			"error": fmt.Sprintf("User can have maximum %d teams.", team.MAX_TEAMS_PER_USER),
		})
	}

	newTeam := &team.Team{
		Name: data.Name,
		Slug: slug.Make(data.Name),
	}

	member := &team.Member{
		UserID: req.Token.UserID,
		Role:   team.ROLE_OWNER,
		Status: true,
	}

	if err := store.CreateTeam(req.Context(), newTeam, member); err != nil {
		return shttp.Error(err)
	}

	newTeam.CurrentUserRole = team.ROLE_OWNER

	return &shttp.Response{
		Status: http.StatusCreated,
		Data:   map[string]any{"team": newTeam.ToMap()},
	}
}
