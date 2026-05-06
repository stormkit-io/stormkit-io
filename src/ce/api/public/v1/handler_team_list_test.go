package publicapiv1_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gosimple/slug"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type HandlerTeamListSuite struct {
	suite.Suite
	*factory.Factory

	conn  databasetest.TestDB
	teams []team.Team
}

func (s *HandlerTeamListSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	config.SetIsSelfHosted(true)
}

func (s *HandlerTeamListSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerTeamListSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

func (s *HandlerTeamListSuite) mockRequest(keyValue string) shttptest.Response {
	return shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		"/v1/teams",
		nil,
		map[string]string{
			"Authorization": keyValue,
		},
	)
}

func (s *HandlerTeamListSuite) Test_Forbidden() {
	response := s.mockRequest("SK_invalid-key")
	s.Equal(http.StatusForbidden, response.Code)
}

func (s *HandlerTeamListSuite) Test_Success_CE() {
	usr := s.MockUser()
	key := s.MockAPIKey(nil, nil, map[string]any{
		"UserID": usr.ID,
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"TeamID": types.ID(0),
		"Scope":  apikey.SCOPE_USER,
	})

	response := s.mockRequest(key.Value)

	s.Equal(http.StatusOK, response.Code)

	expected := fmt.Sprintf(`{"teams": [
		{ "id": "%d", "name": "%s", "slug": "%s", "currentUserRole": "owner", "isDefault": true }
	]}`,
		usr.DefaultTeamID, team.DEFAULT_TEAM_NAME, slug.Make(team.DEFAULT_TEAM_NAME),
	)

	s.JSONEq(expected, response.String())
}

func (s *HandlerTeamListSuite) Test_Success_EE() {
	admin.SetMockLicense()

	usr := s.MockUser()
	store := team.NewStore()
	team1 := team.Team{Name: "My Team", Slug: "my-team"}
	team2 := team.Team{Name: "My Team", Slug: "my-team"}

	s.NoError(store.CreateTeam(context.Background(), &team1, &team.Member{UserID: usr.ID, Role: team.ROLE_OWNER, Status: true}))
	s.NoError(store.CreateTeam(context.Background(), &team2, &team.Member{UserID: usr.ID, Role: team.ROLE_OWNER, Status: true}))

	key := s.MockAPIKey(nil, nil, map[string]any{
		"UserID": usr.ID,
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"TeamID": types.ID(0),
		"Scope":  apikey.SCOPE_USER,
	})

	response := s.mockRequest(key.Value)

	s.Equal(http.StatusOK, response.Code)

	expected := fmt.Sprintf(`{"teams": [
		{ "id": "%d", "name": "%s", "slug": "%s-%d", "currentUserRole": "owner", "isDefault": true },
		{ "id": "%d", "name": "%s", "slug": "%s-%d", "currentUserRole": "owner", "isDefault": false },
		{ "id": "%d", "name": "%s", "slug": "%s-%d", "currentUserRole": "owner", "isDefault": false }
	]}`,
		usr.DefaultTeamID, team.DEFAULT_TEAM_NAME, slug.Make(team.DEFAULT_TEAM_NAME), usr.DefaultTeamID,
		team1.ID, team1.Name, team1.Slug, team1.ID,
		team2.ID, team2.Name, team2.Slug, team2.ID,
	)

	s.JSONEq(expected, response.String())
}

func TestHandlerTeamListSuite(t *testing.T) {
	suite.Run(t, new(HandlerTeamListSuite))
}
