package team_test

import (
	"context"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type TeamStoreSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *TeamStoreSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *TeamStoreSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
}

func (s *TeamStoreSuite) Test_DefaultTeam() {
	store := user.NewStore()
	user, err := store.MustUser(&oauth.User{
		Emails: []oauth.Email{{Address: "test@stormkit.io", IsPrimary: true, IsVerified: true}},
	})

	s.NoError(err)
	s.NotNil(user)
	s.Equal(types.ID(1), user.ID)
	s.Equal("test@stormkit.io", user.PrimaryEmail())

	t, err := team.NewStore().DefaultTeam(context.Background(), user.ID)

	s.NoError(err)
	s.NotNil(t)
	s.Equal(types.ID(1), t.ID)
	s.Equal(team.DEFAULT_TEAM_NAME, t.Name)
	s.Equal(team.ROLE_OWNER, t.CurrentUserRole)
}

func (s *TeamStoreSuite) Test_UserTeams() {
	store := team.NewStore()
	user := s.MockUser(nil)
	newTeam := &team.Team{Name: "My Awesome Team"}
	member := &team.Member{
		Role:   team.ROLE_OWNER,
		UserID: user.ID,
		Status: true,
	}

	s.NoError(store.CreateTeam(context.Background(), newTeam, member))
	s.Equal(newTeam.ID, member.TeamID)
	s.Greater(newTeam.ID, int64(0))

	teams, err := store.Teams(context.Background(), user.ID)
	s.NoError(err)
	s.Len(teams, 2) // Default team and newly created team

	s.Equal(team.DEFAULT_TEAM_NAME, teams[0].Name)
	s.Equal(team.ROLE_OWNER, teams[0].CurrentUserRole)

	s.Equal(newTeam.Name, teams[1].Name)
	s.Equal(team.ROLE_OWNER, teams[1].CurrentUserRole)
}

func TestStoreSuite(t *testing.T) {
	suite.Run(t, &TeamStoreSuite{})
}
