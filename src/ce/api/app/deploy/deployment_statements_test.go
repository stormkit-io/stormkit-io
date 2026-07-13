package deploy_test

import (
	"context"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type DeploymentStatementsSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	usr  *factory.MockUser
	app  *factory.MockApp
	env  *factory.MockEnv
}

func (s *DeploymentStatementsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.usr = s.MockUser(nil)
	s.app = s.MockApp(s.usr, nil)
	s.env = s.MockEnv(s.app, nil)
}

func (s *DeploymentStatementsSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
}

func (s *DeploymentStatementsSuite) Test_TimeoutDeployment() {
	// We need to enable Stormkit Cloud mode to test user metrics update
	config.SetIsStormkitCloud(true)
	defer config.SetIsStormkitCloud(false)

	t1 := time.Now().Add(-30 * time.Minute)
	t2 := time.Now().Add(-60 * time.Minute)
	ctx := context.Background()

	deployment1 := s.MockDeployment(s.env, map[string]any{"CreatedAt": utils.UnixFrom(t1)})
	deployment2 := s.MockDeployment(s.env, map[string]any{"CreatedAt": utils.UnixFrom(t2), "ExitCode": null.IntFrom(1)})

	s.NoError(deploy.NewStore().TimeoutDeployment(ctx, deployment1.ID))
	s.NoError(deploy.NewStore().TimeoutDeployment(ctx, deployment2.ID))

	// This should be timed out because it had no exit code
	d1, err := deploy.NewStore().DeploymentByID(ctx, deployment1.ID)
	s.NoError(err)
	s.Equal(int64(-2), d1.ExitCode.ValueOrZero())

	// This should not be timed out because it had exit code
	d2, err := deploy.NewStore().DeploymentByID(ctx, deployment2.ID)
	s.NoError(err)
	s.Equal(int64(1), d2.ExitCode.ValueOrZero())

	// Now let's verify metrics: should be 30 because only deployment1 was timed out
	metrics, err := user.NewStore().UserMetrics(context.Background(), user.UserMetricsArgs{UserID: s.usr.ID})
	s.NoError(err)
	s.NotNil(metrics)
	s.Equal(int64(30), metrics.BuildMinutes)
}

func TestDeploymentStatementsSuite(t *testing.T) {
	suite.Run(t, new(DeploymentStatementsSuite))
}
