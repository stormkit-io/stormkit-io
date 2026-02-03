package teamhandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gosimple/slug"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team/teamhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerTeamsGetSuite struct {
	suite.Suite
	*factory.Factory
	conn  databasetest.TestDB
	user  *factory.MockUser
	teams []team.Team
}

func (s *HandlerTeamsGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	user := s.MockUser()
	store := team.NewStore()
	team1 := team.Team{Name: "My Team", Slug: "my-team"}
	team2 := team.Team{Name: "My Team", Slug: "my-team"}

	s.NoError(store.CreateTeam(context.Background(), &team1, &team.Member{UserID: user.ID, Role: team.ROLE_OWNER, Status: true}))
	s.NoError(store.CreateTeam(context.Background(), &team2, &team.Member{UserID: user.ID, Role: team.ROLE_OWNER, Status: true}))

	s.user = user
	s.teams = []team.Team{team1, team2}

	config.SetIsSelfHosted(true)
}

func (s *HandlerTeamsGetSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerTeamsGetSuite) mockRequest() shttptest.Response {
	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/teams",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.user.ID),
		},
	)
}

func (s *HandlerTeamsGetSuite) Test_Success_EE() {
	admin.SetMockLicense()

	response := s.mockRequest()
	expected := fmt.Sprintf(`[
		{ "id": "%d", "name": "%s", "slug": "%s-%d", "currentUserRole": "owner", "isDefault": true },
		{ "id": "%d", "name": "%s", "slug": "%s-%d", "currentUserRole": "owner", "isDefault": false },
		{ "id": "%d", "name": "%s", "slug": "%s-%d","currentUserRole": "owner", "isDefault": false }
	]`,
		s.user.DefaultTeamID, team.DEFAULT_TEAM_NAME, slug.Make(team.DEFAULT_TEAM_NAME), s.user.DefaultTeamID,
		s.teams[0].ID, s.teams[0].Name, s.teams[0].Slug, s.teams[0].ID,
		s.teams[1].ID, s.teams[1].Name, s.teams[1].Slug, s.teams[1].ID,
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerTeamsGetSuite) Test_Success_CE() {
	response := s.mockRequest()
	expected := fmt.Sprintf(`[
		{ "id": "%d", "name": "%s", "slug": "%s", "currentUserRole": "owner", "isDefault": true }
	]`,
		s.user.DefaultTeamID, team.DEFAULT_TEAM_NAME, slug.Make(team.DEFAULT_TEAM_NAME),
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func TestHandlerTeamsGet(t *testing.T) {
	suite.Run(t, &HandlerTeamsGetSuite{})
}
