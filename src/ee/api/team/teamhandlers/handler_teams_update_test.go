package teamhandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team/teamhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerTeamsUpdateSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerTeamsUpdateSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerTeamsUpdateSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerTeamsUpdateSuite) TestUpdateTeam_Success() {
	user := s.MockUser()
	newTeam := team.Team{Name: "My Awesome Team"}
	member := team.Member{UserID: user.ID, Role: team.ROLE_OWNER, Status: true}

	s.NoError(team.NewStore().CreateTeam(context.Background(), &newTeam, &member))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodPatch,
		"/team",
		map[string]string{
			"name":   "My Superb Team",
			"teamId": newTeam.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(user.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	myTeam, err := team.NewStore().Team(context.Background(), newTeam.ID, user.ID)
	s.NoError(err)
	s.Equal("My Superb Team", myTeam.Name)
	s.Equal("my-superb-team", myTeam.Slug)
}

func (s *HandlerTeamsUpdateSuite) TestUpdateTeam_ErrDefaultTeam() {
	user := s.MockUser()
	teamID, err := team.NewStore().DefaultTeamID(context.Background(), user.ID)

	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodPatch,
		"/team",
		map[string]string{
			"name":   "My Superb Team",
			"teamId": teamID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(user.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{ "error": "You cannot modify a default team." }`, response.String())
}

func TestHandlerTeamsUpdate(t *testing.T) {
	suite.Run(t, &HandlerTeamsUpdateSuite{})
}
