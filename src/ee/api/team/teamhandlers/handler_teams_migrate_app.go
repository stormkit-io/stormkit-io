package teamhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type TeamMigrateApp struct {
	AppID  types.ID `json:"appId,string"`
	TeamID types.ID `json:"teamId,string"`
}

// handlerTeamsMigrateApp migrates the given app `AppID` to the given `TeamID`.
// The user has to own write access in both teams in order to complete this operation.
func handlerTeamsMigrateApp(req *user.RequestContext) *shttp.Response {
	data := TeamMigrateApp{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	myApp, err := app.NewStore().AppByID(req.Context(), data.AppID)

	if err != nil {
		return shttp.Error(err)
	}

	if myApp == nil {
		return shttp.NotAllowed()
	}

	store := team.NewStore()
	sourceTeam, err := store.Team(req.Context(), data.TeamID, req.User.ID)

	if err != nil {
		return shttp.Error(err)
	}

	if sourceTeam == nil || !team.HasWriteAccess(sourceTeam.CurrentUserRole) {
		return shttp.NotAllowed()
	}

	destTeam, err := store.Team(req.Context(), data.TeamID, req.User.ID)

	if err != nil {
		return shttp.Error(err)
	}

	if destTeam == nil || !team.HasWriteAccess(destTeam.CurrentUserRole) {
		return shttp.NotAllowed()
	}

	if err := store.MigrateApp(req.Context(), data.AppID, destTeam.ID); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
