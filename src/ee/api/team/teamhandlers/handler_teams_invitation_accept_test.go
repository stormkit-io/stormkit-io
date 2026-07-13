package teamhandlers_test

import (
	"context"
	"net/http"
	"testing"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team/teamhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerTeamsInvitationAcceptSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerTeamsInvitationAcceptSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerTeamsInvitationAcceptSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerTeamsInvitationAcceptSuite) TestAcceptInvitation_Success() {
	user1 := s.MockUser()
	user2 := s.MockUser(map[string]any{
		"Emails": []oauth.Email{
			{Address: "hello@stormkit.io", IsVerified: false, IsPrimary: false},
			{Address: "hello-2@stormkit.io", IsVerified: true, IsPrimary: true},
		},
	})

	store := team.NewStore()
	newTeam := team.Team{Name: "My Awesome Team"}
	member := team.Member{UserID: user1.ID, Role: team.ROLE_OWNER, Status: true}

	s.NoError(store.CreateTeam(context.Background(), &newTeam, &member))

	token, err := user.JWT(jwt.MapClaims{
		"inviterId": user1.ID.String(),
		"teamId":    newTeam.ID.String(),
		"email":     user2.PrimaryEmail(),
		"role":      team.ROLE_ADMIN,
	})

	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/team/enroll",
		map[string]string{
			"token": token,
		},
		map[string]string{
			"Authorization": usertest.Authorization(user2.ID),
		},
	)

	teams, err := store.Teams(context.Background(), user2.ID)
	s.Equal(http.StatusOK, response.Code)
	s.NoError(err)
	s.Len(teams, 2)
	s.Equal(teams[1].ID, newTeam.ID)
	s.Equal(teams[1].Name, newTeam.Name)
	s.Equal(teams[1].CurrentUserRole, team.ROLE_ADMIN)
}

func (s *HandlerTeamsInvitationAcceptSuite) TestAcceptInvitation_FailInvalidEmail() {
	user1 := s.MockUser()
	user2 := s.MockUser(map[string]any{
		"Emails": []oauth.Email{{Address: "hello@stormkit.io", IsVerified: true, IsPrimary: true}},
	})

	newTeam := team.Team{Name: "My Awesome Team"}

	token, err := user.JWT(jwt.MapClaims{
		"inviterId": user1.ID.String(),
		"teamId":    newTeam.ID.String(),
		"email":     "random@mail.com",
		"role":      team.ROLE_ADMIN,
	})

	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/team/enroll",
		map[string]string{
			"token": token,
		},
		map[string]string{
			"Authorization": usertest.Authorization(user2.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *HandlerTeamsInvitationAcceptSuite) TestAcceptInvitation_FailInvalidRole() {
	user1 := s.MockUser()
	user2 := s.MockUser(map[string]any{
		"Emails": []oauth.Email{
			{Address: "hello@stormkit.io", IsVerified: true, IsPrimary: true},
		},
	})

	token, err := user.JWT(jwt.MapClaims{
		"inviterId": user1.ID.String(),
		"email":     user2.PrimaryEmail(),
		"role":      "invalid-role",
	})

	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/team/enroll",
		map[string]string{
			"token": token,
		},
		map[string]string{
			"Authorization": usertest.Authorization(user2.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{ "error": "Invalid role given: invalid-role" }`, response.String())
}

func TestHandlerTeamsInvitationAccept(t *testing.T) {
	suite.Run(t, &HandlerTeamsInvitationAcceptSuite{})
}
