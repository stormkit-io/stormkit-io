package jobs_test

import (
	"context"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stretchr/testify/suite"
)

type JobTeamsSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *JobTeamsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *JobTeamsSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
}

func (s *JobTeamsSuite) Test_MarkAppAndEnvAsSoftDeleted() {
	usr := s.MockUser()
	apl := s.MockApp(usr)
	env := s.MockEnv(apl)
	ctx := context.TODO()

	// Make sure the second app and env is not deleted
	usr2 := s.MockUser()
	apl2 := s.MockApp(usr2)
	env2 := s.MockEnv(apl2)

	// Delete the team
	s.conn.Exec("UPDATE teams SET deleted_at = NOW() WHERE team_id = $1;", usr.DefaultTeamID)

	s.NoError(jobs.CleanupDeletedTeams(ctx))

	deletedEnv, err := buildconf.NewStore().EnvironmentByID(ctx, env.ID)
	s.NoError(err)
	s.Nil(deletedEnv)

	deletedApp, err := app.NewStore().AppByID(ctx, apl.ID)
	s.NoError(err)
	s.Nil(deletedApp)

	notDeletedEnv, err := buildconf.NewStore().EnvironmentByID(ctx, env2.ID)
	s.NoError(err)
	s.NotNil(notDeletedEnv)

	notDeletedApp, err := app.NewStore().AppByID(ctx, apl2.ID)
	s.NoError(err)
	s.NotNil(notDeletedApp)
}

func TestJobTeams(t *testing.T) {
	suite.Run(t, &JobTeamsSuite{})
}
