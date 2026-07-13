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

type HandlerTeamStatsTopDomainsSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerTeamStatsTopDomainsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerTeamStatsTopDomainsSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerTeamStatsTopDomainsSuite) Test_Success() {
	usr := s.MockUser()
	newTeam := team.Team{Name: "My Awesome Team"}
	member := team.Member{UserID: usr.ID, Role: team.ROLE_OWNER, Status: true}

	s.NoError(team.NewStore().CreateTeam(context.Background(), &newTeam, &member))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/team/stats/domains?teamId=%d", newTeam.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{
		"domains": null
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerTeamStatsTopDomainsSuite) Test_NotAllowed_NonMember() {
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

func (s *HandlerTeamStatsTopDomainsSuite) Test_BadRequest_InvalidTeamId() {
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

func (s *HandlerTeamStatsTopDomainsSuite) Test_BadRequest_MissingTeamId() {
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

func (s *HandlerTeamStatsTopDomainsSuite) Test_Unauthorized_NoAuthHeader() {
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

func TestHandlerTeamStatsDomains(t *testing.T) {
	suite.Run(t, &HandlerTeamStatsTopDomainsSuite{})
}
