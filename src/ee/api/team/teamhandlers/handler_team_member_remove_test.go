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

type HandlerTeamMemberRemoveSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerTeamMemberRemoveSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerTeamMemberRemoveSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerTeamMemberRemoveSuite) Test_Success() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()
	store := team.NewStore()
	newTeam := team.Team{Name: "My Awesome Team"}
	member1 := team.Member{UserID: usr1.ID, Role: team.ROLE_OWNER, Status: true}
	member2 := team.Member{UserID: usr2.ID, Role: team.ROLE_DEVELOPER, Status: false}

	s.NoError(store.CreateTeam(context.Background(), &newTeam, &member1))

	member2.TeamID = newTeam.ID
	s.NoError(store.AddMemberToTeam(context.Background(), &member2))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/team/member?teamId=%d&memberId=%d", newTeam.ID, member2.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr1.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	members, err := store.TeamMembers(context.Background(), team.TeamMemberFilters{
		TeamID: newTeam.ID,
	})

	s.NoError(err)
	s.Len(members, 1)
}

func (s *HandlerTeamMemberRemoveSuite) Test_RemovingOwner() {
	usr := s.MockUser()
	store := team.NewStore()
	newTeam := team.Team{Name: "My Awesome Team"}
	member := team.Member{UserID: usr.ID, Role: team.ROLE_OWNER, Status: true}

	s.NoError(store.CreateTeam(context.Background(), &newTeam, &member))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/team/member?teamId=%d&memberId=%d", newTeam.ID, member.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{ "error": "Cannot remove the only owner of this team. Delete the team instead." }`, response.String())
}

func TestHandlerTeamMemberRemove(t *testing.T) {
	suite.Run(t, &HandlerTeamMemberRemoveSuite{})
}
