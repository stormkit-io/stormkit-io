package deployservice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"
	"github.com/stormkit-io/stormkit-io/src/lib/tasks"
	"github.com/stretchr/testify/suite"
)

type RepoSizeSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *RepoSizeSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	insp := tasks.Inspector()
	insp.DeleteQueue(tasks.QueueDeployService, true)
}

func (s *RepoSizeSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	config.SetIsStormkitCloud(false)
	deployservice.MockRepoSize = nil
	s.T().Setenv(config.MaxRepoSizeEnvVar, "")
}

// deployWithSize runs a deployment while the provider reports size bytes for
// the repository, and returns whatever Deploy came back with.
func (s *RepoSizeSuite) deployWithSize(size int64, err error) error {
	deployservice.MockRepoSize = func(*app.App) (int64, error) {
		return size, err
	}

	usr := s.MockUser()
	mockApp := s.MockApp(usr)

	deployer := &deployservice.DefaultDeployer{}

	return deployer.Deploy(context.Background(), mockApp.App, &deploy.Deployment{
		AppID: mockApp.ID,
	})
}

// assertRejected asserts the deployment was refused for being too large, with
// the message and status the customer receives.
func (s *RepoSizeSuite) assertRejected(err error) {
	s.Require().Error(err)

	var serr *shttperr.Error

	s.Require().True(errors.As(err, &serr), "expected an shttperr, got %T", err)
	s.Equal("repo-too-large", serr.Code())
	s.Equal(422, serr.Status())
	s.Contains(serr.Error(), "Repository is larger than allowed")
}

func (s *RepoSizeSuite) Test_Rejects_OversizedRepo_OnCloud() {
	config.SetIsStormkitCloud(true)

	s.assertRejected(s.deployWithSize(2<<30, nil))
}

func (s *RepoSizeSuite) Test_ReportsBothNumbers() {
	config.SetIsStormkitCloud(true)

	err := s.deployWithSize(3<<30, nil)

	s.Require().Error(err)
	s.Contains(err.Error(), "3.0GB")
	s.Contains(err.Error(), "1.0GB")
}

func (s *RepoSizeSuite) Test_Allows_RepoWithinTheCap() {
	config.SetIsStormkitCloud(true)

	// The deployment continues past the gate; whatever it does afterwards is
	// not this check's business, so only the rejection is ruled out.
	err := s.deployWithSize(deployservice.CloudMaxRepoSize-1, nil)

	if err != nil {
		s.NotContains(err.Error(), "Repository is larger than allowed")
	}
}

func (s *RepoSizeSuite) Test_Allows_RepoExactlyAtTheCap() {
	config.SetIsStormkitCloud(true)

	err := s.deployWithSize(deployservice.CloudMaxRepoSize, nil)

	if err != nil {
		s.NotContains(err.Error(), "Repository is larger than allowed")
	}
}

func (s *RepoSizeSuite) Test_SelfHosted_IsUncapped() {
	config.SetIsStormkitCloud(false)

	err := s.deployWithSize(50<<30, nil)

	if err != nil {
		s.NotContains(err.Error(), "Repository is larger than allowed")
	}
}

// A provider that cannot answer must not take deployments down with it: the
// build host still has its own disk, this check only saves it the trip.
func (s *RepoSizeSuite) Test_ProviderFailure_DoesNotBlockTheDeployment() {
	config.SetIsStormkitCloud(true)

	err := s.deployWithSize(0, errors.New("bitbucket is having a day"))

	if err != nil {
		s.NotContains(err.Error(), "Repository is larger than allowed")
	}
}

// A provider that returns nothing (GitLab hides statistics from tokens below
// Reporter access) reads as 0 and must not reject the deployment.
func (s *RepoSizeSuite) Test_UnknownSize_DoesNotBlockTheDeployment() {
	config.SetIsStormkitCloud(true)

	err := s.deployWithSize(0, nil)

	if err != nil {
		s.NotContains(err.Error(), "Repository is larger than allowed")
	}
}

func (s *RepoSizeSuite) Test_EnvOverride_RaisesTheCap() {
	config.SetIsStormkitCloud(true)
	s.T().Setenv(config.MaxRepoSizeEnvVar, "4096")

	err := s.deployWithSize(3<<30, nil)

	if err != nil {
		s.NotContains(err.Error(), "Repository is larger than allowed")
	}
}

func (s *RepoSizeSuite) Test_EnvOverride_LowersTheCap() {
	config.SetIsStormkitCloud(true)
	s.T().Setenv(config.MaxRepoSizeEnvVar, "100")

	s.assertRejected(s.deployWithSize(200<<20, nil))
}

func (s *RepoSizeSuite) Test_UnparseableOverride_FallsBackToTheDefault() {
	config.SetIsStormkitCloud(true)
	s.T().Setenv(config.MaxRepoSizeEnvVar, "not-a-number")

	s.assertRejected(s.deployWithSize(2<<30, nil))
}

func TestRepoSize(t *testing.T) {
	suite.Run(t, &RepoSizeSuite{})
}
