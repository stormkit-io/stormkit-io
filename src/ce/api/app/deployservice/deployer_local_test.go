package deployservice_test

import (
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/mise"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type DeployerLocalSuite struct {
	suite.Suite
	conn             databasetest.TestDB
	mockExec         *mocks.CommandInterface
	mockMise         *mocks.MiseInterface
	mockDeploymentID string
	mockExecutable   string
}

func (s *DeployerLocalSuite) SetupTest() {
	// runService forwards this when it is set, which would break the exact
	// Env match below on a host that exports it.
	s.T().Setenv(config.MaxRepoSizeEnvVar, "")
	s.NoError(os.Unsetenv(config.MaxRepoSizeEnvVar))

	s.conn = databasetest.InitTx("deployer_local_suite")
	s.mockDeploymentID = "13051616"
	s.mockExecutable = "mock/path"
	config.Get().Deployer.Executable = s.mockExecutable
}

func (s *DeployerLocalSuite) BeforeTest(_, _ string) {
	s.mockExec = &mocks.CommandInterface{}
	s.mockMise = &mocks.MiseInterface{}
	sys.DefaultCommand = s.mockExec
	mise.DefaultMise = s.mockMise
}

func (s *DeployerLocalSuite) AfterTest(_, _ string) {
	os.RemoveAll(path.Join(os.TempDir(), "deployment-"+s.mockDeploymentID))
	sys.DefaultCommand = nil
	mise.DefaultMise = nil
}

func (s *DeployerLocalSuite) TearDownTest() {
	s.conn.Close()
}

func (s *DeployerLocalSuite) Test_StartDeployment() {
	deployer := deployservice.Local()
	payload, err := json.Marshal(map[string]any{
		"baseUrl":       "http://api.stormkit:8888",
		"deploymentId":  s.mockDeploymentID,
		"deploymentMsg": "some-message",
		"rootDir":       path.Join(os.TempDir(), "deployment-"+s.mockDeploymentID),
	})

	s.NoError(err)

	s.mockMise.On("InstallMise", mock.Anything).Return(nil).Once()
	s.mockMise.On("Prune", mock.Anything, mock.Anything).Return(nil).Once()
	s.mockExec.On("SetOpts", sys.CommandOpts{
		Name: s.mockExecutable,
		Args: []string{
			"--payload",
			string(payload),
		},
		Env: []string{
			"PATH=" + os.Getenv("PATH"),
			"HOME=" + os.Getenv("HOME"),
			"STORMKIT_DEPLOYER_DIR=" + config.Get().Deployer.StorageDir,
			"STORMKIT_DEPLOYER_SERVICE=" + config.DeployerServiceLocal,
			"STORMKIT_APP_SECRET=" + config.AppSecret(),
		},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}).Return(s.mockExec).Once()
	s.mockExec.On("Run").Return(nil, nil).Once()

	s.NoError(deployer.SendPayload(deployservice.SendPayloadArgs{
		DeploymentID: utils.StringToID(s.mockDeploymentID),
		EncryptedMsg: "some-message",
	}))
}

func TestDeployerLocalSuite(t *testing.T) {
	suite.Run(t, new(DeployerLocalSuite))
}
