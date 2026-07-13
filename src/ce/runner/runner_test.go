package runner_test

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	"github.com/stormkit-io/stormkit-io/src/ce/runner"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type RunnerSuite struct {
	suite.Suite
	config        runner.RunnerOpts
	mockInstaller mocks.InstallerInterface
	mockBuilder   mocks.BuilderInterface
	mockRepo      mocks.RepoInterface
	mockUploader  mocks.RunnerUploaderInterface
	mockBundler   mocks.BundlerInterface
}

func (s *RunnerSuite) BeforeTest(_, _ string) {
	tmpDir, err := os.MkdirTemp("", "tmp-test-runner-")

	s.NoError(err)

	s.config = runner.RunnerOpts{
		RootDir:  tmpDir,
		Reporter: runner.NewReporter("https://example.com"),
	}

	s.mockInstaller = mocks.InstallerInterface{}
	s.mockBuilder = mocks.BuilderInterface{}
	s.mockRepo = mocks.RepoInterface{}
	s.mockUploader = mocks.RunnerUploaderInterface{}
	s.mockBundler = mocks.BundlerInterface{}

	runner.DefaultBuilder = &s.mockBuilder
	runner.DefaultInstaller = &s.mockInstaller
	runner.DefaultRepo = &s.mockRepo
	runner.DefaultUploader = &s.mockUploader
	runner.DefaultBundler = &s.mockBundler
}

func (s *RunnerSuite) AfterTest(_, _ string) {
	if strings.Contains(s.config.RootDir, os.TempDir()) {
		s.config.RemoveAll()
	}

	runner.DefaultBuilder = nil
	runner.DefaultInstaller = nil
	runner.DefaultRepo = nil
	runner.DefaultUploader = nil
}

func (s *RunnerSuite) Test_Start_InvalidPayload() {
	err := runner.Start("", "")
	s.Error(err)
	s.Equal("unexpected end of JSON input", err.Error())
}

func (s *RunnerSuite) Test_Start_InvaldDecryption() {
	err := runner.Start(`{ "deploymentMsg": "invalid-msg" }`, "")
	s.Error(err)
	s.Equal("illegal base64 data at input byte 8", err.Error())
}

// This test will checkout a sample github repository and build it.
// It uses bun as it's significantly faster.
func (s *RunnerSuite) Test_AllProcess() {
	s.mockRepo.On("Checkout", mock.Anything).Return(nil)
	s.mockRepo.On("CommitInfo").Return(map[string]string{})
	s.mockInstaller.On("InstallRuntimeDependencies", mock.Anything).Return([]string{"go@1.24"}, nil)
	s.mockInstaller.On("RuntimeVersion", mock.Anything).Return(nil)
	s.mockInstaller.On("Install", mock.Anything).Return(nil)
	s.mockBuilder.On("ExecCommands", mock.Anything).Return(nil)
	s.mockBuilder.On("BuildApiIfNecessary", mock.Anything).Return(true, nil)
	s.mockBundler.On("Bundle", mock.Anything).Return(nil)
	s.mockBundler.On("ParseRedirects").Return(nil)
	s.mockBundler.On("ParseHeaders").Return(nil)
	s.mockUploader.On("Upload", runner.UploadArgs{
		ClientZip:    fmt.Sprintf("%s/dist/sk-client.zip", s.config.RootDir),
		AppID:        2501,
		EnvID:        51191,
		DeploymentID: 1234,
		EnvVars:      map[string]string{},
	}).Return(nil, nil)

	msg := &deployservice.DeploymentMessage{
		Client: deployservice.ClientConfig{
			Repo:        "https://github.com/stormkit-dev/e2e-npm",
			AccessToken: "some-token",
		},
		Build: deployservice.BuildConfig{
			Branch: "main",
			AppID:  "2501",
			EnvID:  "51191",
		},
	}

	encryptedMsg, err := msg.Encrypt()

	s.NoError(err)

	payload := runner.Payload{
		DeploymentID:  "1234",
		DeploymentMsg: encryptedMsg,
		RootDir:       s.config.RootDir,
	}

	b, err := json.Marshal(payload)
	s.NoError(err)
	s.NoError(runner.Start(string(b), ""))
}

func TestRunnerSuite(t *testing.T) {
	suite.Run(t, &RunnerSuite{})
}
