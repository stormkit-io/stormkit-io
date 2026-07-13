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

type HandlerTeamMembersSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerTeamMembersSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerTeamMembersSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerTeamMembersSuite) Test_Success() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()
	newTeam := team.Team{Name: "My Awesome Team"}
	member1 := team.Member{UserID: usr1.ID, Role: team.ROLE_OWNER, Status: true}
	member2 := team.Member{UserID: usr2.ID, Role: team.ROLE_DEVELOPER, Status: false}

	s.NoError(team.NewStore().CreateTeam(context.Background(), &newTeam, &member1))

	member2.TeamID = newTeam.ID
	s.NoError(team.NewStore().AddMemberToTeam(context.Background(), &member2))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/team/members?teamId=%d", newTeam.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr1.ID),
		},
	)

	expected := fmt.Sprintf(`[
		{
		  "id": "%d",
		  "teamId": "%d",
		  "displayName": "dlorenzo",
		  "email": "%s",
		  "firstName": "David",
		  "fullName": "David Lorenzo",
		  "lastName": "Lorenzo",
		  "avatar": "",
		  "role": "owner",
		  "status": true,
		  "userId": "%s"
		},
		{
		  "id": "%d",
		  "teamId": "%d",
		  "displayName": "dlorenzo",
		  "email": "%s",
		  "firstName": "David",
		  "fullName": "David Lorenzo",
		  "lastName": "Lorenzo",
		  "avatar": "",
		  "role": "developer",
		  "status": false,
		  "userId": "%s"
		}
	  ]`,
		member1.ID,
		newTeam.ID,
		usr1.PrimaryEmail(),
		usr1.ID.String(),
		member2.ID,
		newTeam.ID,
		usr2.PrimaryEmail(),
		usr2.ID.String(),
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func TestHandlerTeamMembers(t *testing.T) {
	suite.Run(t, &HandlerTeamMembersSuite{})
}
