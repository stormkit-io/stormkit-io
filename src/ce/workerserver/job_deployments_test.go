package jobs_test

import (
	"context"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type JobDeploymentsSuite struct {
	suite.Suite
	*factory.Factory
	conn       databasetest.TestDB
	mockClient mocks.ClientInterface
}

func (s *JobDeploymentsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockClient = mocks.ClientInterface{}
	integrations.SetDefaultClient(&s.mockClient)
}

func (s *JobDeploymentsSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	integrations.SetDefaultClient(nil)
}

func (s *JobDeploymentsSuite) Test_RemoveDeploymentsArtifacts() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	T45daysAgo := utils.NewUnix()
	T45daysAgo.Time = T45daysAgo.AddDate(0, 0, -45)

	T15daysAgo := utils.NewUnix()
	T15daysAgo.Time = T15daysAgo.AddDate(0, 0, -15)

	T60daysAgo := utils.NewUnix()
	T60daysAgo.Time = T60daysAgo.AddDate(0, 0, -60)

	// We have 4 deployments:
	//
	// - A deployment created 45 days ago						(this one should be removed because it is older than 30 days and deleted)
	// - A deployment created 15 days ago and deleted			(this one should be removed because it is deleted)
	// - A deployment created 60 days ago, but still published  (published - so no deletion)
	// - A deployment that is not deleted						(fresh deployment - so no deletion)
	deployments := s.MockDeployments(
		4,
		env,
		map[string]any{"CreatedAt": T45daysAgo, "UploadResult": &deploy.UploadResult{ClientLocation: "local:/d-1"}},
		map[string]any{"CreatedAt": T15daysAgo, "DeletedAt": utils.NewUnix(), "UploadResult": &deploy.UploadResult{ClientLocation: "local:/d-2"}},
		map[string]any{"CreatedAt": T60daysAgo, "Published": deploy.PublishedInfo{{env.ID, 100}}},
	)

	s.mockClient.On("DeleteArtifacts", mock.Anything, integrations.DeleteArtifactsArgs{StorageLocation: "local:/d-2"}).Return(nil).Once()
	s.mockClient.On("DeleteArtifacts", mock.Anything, integrations.DeleteArtifactsArgs{StorageLocation: "local:/d-1"}).Return(nil).Once()

	s.NoError(jobs.RemoveDeploymentArtifacts(context.Background()))

	ids := []types.ID{}

	rows, err := s.conn.QueryContext(
		context.Background(),
		`SELECT deployment_id FROM deployments WHERE artifacts_deleted IS TRUE;`,
	)

	s.NoError(err)
	s.NotNil(rows)

	defer rows.Close()

	for rows.Next() {
		var id types.ID
		s.NoError(rows.Scan(&id))
		ids = append(ids, id)
	}

	s.Len(ids, 2)
	s.Equal(ids[0], deployments[0].ID)
	s.Equal(ids[1], deployments[1].ID)
}

func TestJobDeploymentsSuite(t *testing.T) {
	suite.Run(t, &JobDeploymentsSuite{})
}
