package teamhandlers_test

import (
	"context"
	"fmt"
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

type HandlerTeamStatsSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerTeamStatsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerTeamStatsSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerTeamStatsSuite) Test_Success() {
	usr := s.MockUser()
	newTeam := team.Team{Name: "My Awesome Team"}
	member := team.Member{UserID: usr.ID, Role: team.ROLE_OWNER, Status: true}

	s.NoError(team.NewStore().CreateTeam(context.Background(), &newTeam, &member))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/team/stats?teamId=%d", newTeam.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{
		"totalRequests": {
			"current": 0,
			"previous": 0
		},
		"totalApps": {
			"total": 0,
			"new": 0,
			"deleted": 0
		},
		"totalDeployments": {
			"total": 0,
			"current": 0,
			"previous": 0
		},
		"avgDeploymentDuration": {
			"current": 0,
			"previous": 0
		}
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerTeamStatsSuite) Test_NotAllowed_NonMember() {
	usr := s.MockUser()
	otherUser := s.MockUser()
	newTeam := team.Team{Name: "My Awesome Team"}
	member := team.Member{UserID: usr.ID, Role: team.ROLE_OWNER, Status: true}

	s.NoError(team.NewStore().CreateTeam(context.Background(), &newTeam, &member))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/team/stats?teamId=%d", newTeam.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(otherUser.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *HandlerTeamStatsSuite) Test_BadRequest_InvalidTeamId() {
	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/team/stats?teamId=invalid",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *HandlerTeamStatsSuite) Test_BadRequest_MissingTeamId() {
	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/team/stats",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *HandlerTeamStatsSuite) Test_Unauthorized_NoAuthHeader() {
	usr := s.MockUser()
	newTeam := team.Team{Name: "My Awesome Team"}
	member := team.Member{UserID: usr.ID, Role: team.ROLE_OWNER, Status: true}

	s.NoError(team.NewStore().CreateTeam(context.Background(), &newTeam, &member))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/team/stats?teamId=%d", newTeam.ID),
		nil,
		map[string]string{},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func TestHandlerTeamStats(t *testing.T) {
	suite.Run(t, &HandlerTeamStatsSuite{})
}
