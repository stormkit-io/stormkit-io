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

type HandlerTeamsDeleteSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerTeamsDeleteSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerTeamsDeleteSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerTeamsDeleteSuite) TestDeleteTeam_Success() {
	user := s.MockUser()
	store := team.NewStore()
	myTeam := team.Team{Name: "My Second Team"}
	member := team.Member{UserID: user.ID, Role: team.ROLE_OWNER, Status: true}

	s.NoError(store.CreateTeam(context.Background(), &myTeam, &member))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/team?teamId=%d", myTeam.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(user.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	t, err := store.Team(context.Background(), myTeam.ID, user.ID)
	s.NoError(err)
	s.Nil(t)
}

func (s *HandlerTeamsDeleteSuite) TestDeleteTeam_ErrDefaultTeam() {
	user := s.MockUser()
	store := team.NewStore()
	teamID, err := store.DefaultTeamID(context.Background(), user.ID)

	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/team?teamId=%d", teamID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(user.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"You cannot delete a default team."}`, response.String())
}

func (s *HandlerTeamsDeleteSuite) TestDeleteTeam_ErrPermission() {
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
		shttp.MethodDelete,
		fmt.Sprintf("/team?teamId=%d", myTeam.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(user2.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func TestHandlerTeamsDelete(t *testing.T) {
	suite.Run(t, &HandlerTeamsDeleteSuite{})
}
