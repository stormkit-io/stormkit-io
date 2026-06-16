package teamhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services sets the handlers for this service.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	s.NewEndpoint("/team").
		Middleware(user.WithEE).
		Handler(shttp.MethodDelete, "", user.WithAuth(handlerTeamsDelete)).
		Handler(shttp.MethodPatch, "", user.WithAuth(handlerTeamsUpdate)).
		Handler(shttp.MethodPost, "/invite", user.WithAuth(handlerTeamsInvite)).
		Handler(shttp.MethodPost, "/enroll", user.WithAuth(handlerTeamsInvitationAccept)).
		Handler(shttp.MethodGet, "/members", user.WithAuth(handlerTeamMembers)).
		Handler(shttp.MethodDelete, "/member", user.WithAuth(handlerTeamMemberRemove)).
		Handler(shttp.MethodPost, "/migrate", user.WithAuth(handlerTeamsMigrateApp)).
		Handler(shttp.MethodGet, "/stats", user.WithAuth(handlerTeamStats)).
		Handler(shttp.MethodGet, "/stats/domains", user.WithAuth(handleTeamStatsDomains))

	return s
}
