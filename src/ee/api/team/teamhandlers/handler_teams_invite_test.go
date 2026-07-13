package teamhandlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
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

type HandlerTeamsInviteSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerTeamsInviteSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerTeamsInviteSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerTeamsInviteSuite) TestInviteNewMember_Success() {
	usr := s.MockUser()
	newTeam := team.Team{Name: "My Awesome Team"}
	member := team.Member{UserID: usr.ID, Role: team.ROLE_OWNER, Status: true}

	s.NoError(team.NewStore().CreateTeam(context.Background(), &newTeam, &member))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/team/invite",
		map[string]string{
			"email":  "test@stormkit.io",
			"teamId": newTeam.ID.String(),
			"role":   team.ROLE_DEVELOPER,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	resp := map[string]string{}

	s.Equal(http.StatusOK, response.Code)
	s.NoError(json.Unmarshal(response.Byte(), &resp))

	claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: resp["token"]})
	s.Equal("test@stormkit.io", claims["email"])
	s.Equal(newTeam.ID.String(), claims["teamId"])
	s.Equal(usr.ID.String(), claims["inviterId"])
}

func (s *HandlerTeamsInviteSuite) TestInviteNewMember_ErrDefaultTeam() {
	usr := s.MockUser()
	teamID, err := team.NewStore().DefaultTeamID(context.Background(), usr.ID)
	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/team/invite",
		map[string]string{
			"email":  "test@stormkit.io",
			"teamId": teamID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{ "error": "You cannot add a member to your default team. Please create a new team." }`

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerTeamsInviteSuite) TestInviteNewMember_ErrPermission() {
	user1 := s.MockUser()
	user2 := s.MockUser()

	store := team.NewStore()

	myTeam := team.Team{Name: "My Other Team"}
	member1 := team.Member{UserID: user1.ID, Role: team.ROLE_OWNER, Status: true}
	member2 := team.Member{UserID: user2.ID, Role: team.ROLE_DEVELOPER, Status: true}

	s.NoError(store.CreateTeam(context.Background(), &myTeam, &member1))

	member2.TeamID = myTeam.ID

	s.NoError(store.AddMemberToTeam(context.Background(), &member2))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/team/invite",
		map[string]string{
			"email":  "test@stormkit.io",
			"teamId": myTeam.ID.String(),
			"role":   team.ROLE_DEVELOPER,
		},
		map[string]string{
			"Authorization": usertest.Authorization(user2.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func TestHandlerTeamsInvite(t *testing.T) {
	suite.Run(t, &HandlerTeamsInviteSuite{})
}
