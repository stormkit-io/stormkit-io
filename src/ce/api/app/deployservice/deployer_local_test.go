package deployservice_test

import (
	"encoding/json"
	"os"
	"os/exec"
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

// Test_CleansWorkspaceAfterProcessExit exercises real child exits without the
// runner's own deferred cleanup, including a child killed before it can run it.
func (s *DeployerLocalSuite) Test_CleansWorkspaceAfterProcessExit() {
	for _, tc := range []struct {
		name   string
		script string
		failed bool
	}{
		{name: "success", script: "exit 0"},
		{name: "failure", script: "exit 23", failed: true},
		{name: "killed", script: "kill -KILL $$", failed: true},
	} {
		s.Run(tc.name, func() {
			s.T().Setenv("TMPDIR", s.T().TempDir())
			rootDir := path.Join(os.TempDir(), "deployment-"+s.mockDeploymentID)
			sibling := path.Join(os.TempDir(), "deployment-13051617", "repo")
			s.Require().NoError(os.MkdirAll(sibling, 0700))

			executable := path.Join(s.T().TempDir(), "deployer")
			s.Require().NoError(os.WriteFile(executable, []byte("#!/bin/sh\n"+tc.script+"\n"), 0700))
			config.Get().Deployer.Executable = executable
			sys.DefaultCommand = nil

			err := deployservice.Local().SendPayload(deployservice.SendPayloadArgs{
				DeploymentID: utils.StringToID(s.mockDeploymentID),
				EncryptedMsg: "unused-by-test-child",
			})

			if tc.failed {
				var exitErr *exec.ExitError
				s.ErrorAs(err, &exitErr)
			} else {
				s.NoError(err)
			}

			s.NoDirExists(rootDir)
			s.DirExists(sibling)
		})
	}
}

func (s *DeployerLocalSuite) Test_CleansWorkspaceWhenProcessCannotStart() {
	s.T().Setenv("TMPDIR", s.T().TempDir())
	rootDir := path.Join(os.TempDir(), "deployment-"+s.mockDeploymentID)
	config.Get().Deployer.Executable = path.Join(s.T().TempDir(), "missing-deployer")
	sys.DefaultCommand = nil

	err := deployservice.Local().SendPayload(deployservice.SendPayloadArgs{
		DeploymentID: utils.StringToID(s.mockDeploymentID),
		EncryptedMsg: "unused-by-test-child",
	})

	s.ErrorIs(err, os.ErrNotExist)
	s.NoDirExists(rootDir)
}

func (s *DeployerLocalSuite) Test_KeepsWorkspaceUntilProcessReturns() {
	s.T().Setenv("TMPDIR", s.T().TempDir())
	rootDir := path.Join(os.TempDir(), "deployment-"+s.mockDeploymentID)
	s.mockExec.On("SetOpts", mock.Anything).Return(s.mockExec).Once()
	s.mockExec.On("Run").Run(func(mock.Arguments) {
		s.DirExists(path.Join(rootDir, "repo"))
		s.DirExists(path.Join(rootDir, "keys"))
		s.Require().NoError(os.MkdirAll(path.Join(rootDir, "dist"), 0700))
		s.Require().NoError(os.WriteFile(path.Join(rootDir, "dist", "server.zip"), []byte("artifact"), 0600))
	}).Return(nil).Once()

	s.NoError(deployservice.Local().SendPayload(deployservice.SendPayloadArgs{
		DeploymentID: utils.StringToID(s.mockDeploymentID),
		EncryptedMsg: "unused-by-test-child",
	}))

	s.NoDirExists(rootDir)
	s.mockExec.AssertExpectations(s.T())
}

func TestDeployerLocalSuite(t *testing.T) {
	suite.Run(t, new(DeployerLocalSuite))
}
