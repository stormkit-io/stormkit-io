package teamhandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team/teamhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerTeamsMigrateAppSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerTeamsMigrateAppSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerTeamsMigrateAppSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerTeamsMigrateAppSuite) Test_Success() {
	user := s.MockUser()
	myApp := s.MockApp(user)
	newTeam := team.Team{Name: "My Awesome Team"}
	member := team.Member{UserID: user.ID, Role: team.ROLE_OWNER, Status: true}

	s.NoError(team.NewStore().CreateTeam(context.Background(), &newTeam, &member))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/team/migrate",
		map[string]string{
			"appId":  myApp.ID.String(),
			"teamId": newTeam.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(user.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	updatedApp, err := app.NewStore().AppByID(context.Background(), myApp.ID)
	s.NoError(err)
	s.Equal(newTeam.ID, updatedApp.TeamID)
}

func (s *HandlerTeamsMigrateAppSuite) Test_FailPermission() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()
	myApp := s.MockApp(usr2)
	newTeam := team.Team{Name: "My Awesome Team"}
	member := team.Member{UserID: usr1.ID, Role: team.ROLE_OWNER, Status: true}

	s.NoError(team.NewStore().CreateTeam(context.Background(), &newTeam, &member))
	s.NoError(team.NewStore().AddMemberToTeam(context.Background(), &team.Member{
		TeamID: newTeam.ID,
		UserID: usr2.ID,
		Role:   team.ROLE_DEVELOPER,
		Status: true,
	}))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/team/migrate",
		map[string]string{
			"appId":  myApp.ID.String(),
			"teamId": newTeam.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr2.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

// Test_FailNotMemberOfSourceTeam verifies that an app cannot be moved into the
// caller's team unless the caller has write access to the app's current team.
func (s *HandlerTeamsMigrateAppSuite) Test_FailNotMemberOfSourceTeam() {
	victim := s.MockUser()
	victimApp := s.MockApp(victim)

	attacker := s.MockUser()
	attackerTeam := team.Team{Name: "Attacker Team"}
	member := team.Member{UserID: attacker.ID, Role: team.ROLE_OWNER, Status: true}
	s.Require().NoError(team.NewStore().CreateTeam(context.Background(), &attackerTeam, &member))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(teamhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/team/migrate",
		map[string]string{
			"appId":  victimApp.ID.String(),
			"teamId": attackerTeam.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(attacker.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)

	unchanged, err := app.NewStore().AppByID(context.Background(), victimApp.ID)
	s.Require().NoError(err)
	s.Equal(victimApp.TeamID, unchanged.TeamID)
}

func TestHandlerTeamsMigrateApp(t *testing.T) {
	suite.Run(t, &HandlerTeamsMigrateAppSuite{})
}
