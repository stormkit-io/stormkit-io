package runner_test

import (
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/runner"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type UploaderSuite struct {
	suite.Suite
	config           runner.RunnerOpts
	mockIntegrations *mocks.ClientInterface
}

type MockFileInfo struct {
	name string
	size int64
}

func (m MockFileInfo) Name() string       { return m.name }
func (m MockFileInfo) Size() int64        { return m.size }
func (m MockFileInfo) Mode() os.FileMode  { return os.ModePerm }
func (m MockFileInfo) ModTime() time.Time { return time.Now() }
func (m MockFileInfo) IsDir() bool        { return false }
func (m MockFileInfo) Sys() any           { return nil }

func (s *UploaderSuite) BeforeTest(_, _ string) {
	tmpDir, err := os.MkdirTemp("", "tmp-test-runner-")

	s.NoError(err)

	s.config = runner.RunnerOpts{
		RootDir: tmpDir,
		KeysDir: path.Join(tmpDir, "keys"),
		Repo: runner.RepoOpts{
			Dir: path.Join(tmpDir, "repo"),
		},
	}

	s.NoError(s.config.MkdirAll())

	s.mockIntegrations = &mocks.ClientInterface{}
	integrations.SetDefaultClient(s.mockIntegrations)

	s.NoError(os.MkdirAll(path.Join(s.config.RootDir, "dist"), 0774))
	s.NoError(os.MkdirAll(path.Join(s.config.Repo.Dir, ".stormkit", "public"), 0774))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, ".stormkit", "public", "index.html"), []byte("Hello world"), 0664))
	s.NoError(file.ZipV2(file.ZipArgs{
		Source:  []string{path.Join(s.config.Repo.Dir, ".stormkit", "public")},
		ZipName: path.Join(s.config.RootDir, "dist", "sk-client.zip"),
	}))
}

func (s *UploaderSuite) AfterTest(_, _ string) {
	if strings.Contains(s.config.RootDir, os.TempDir()) {
		s.config.RemoveAll()
	}

	integrations.SetDefaultClient(nil)
}

// This test will checkout a sample github repository and build it.
// It uses bun as it's significantly faster.
func (s *UploaderSuite) Test_Upload() {
	s.mockIntegrations.On("Upload", integrations.UploadArgs{
		MigrationsZip: path.Join(s.config.RootDir, "dist", "sk-migrations.zip"),
		ClientZip:     path.Join(s.config.RootDir, "dist", "sk-client.zip"),
		AppID:         2501,
		EnvID:         51191,
		DeploymentID:  202521,
	}).Return(&integrations.UploadResult{
		Migrations: integrations.UploadOverview{
			FilesUploaded: 1,
			BytesUploaded: 512,
		},
		Client: integrations.UploadOverview{
			FilesUploaded: 15,
			BytesUploaded: 4102,
		},
	}, nil)

	result, err := runner.NewUploader(s.config.Uploader).Upload(runner.UploadArgs{
		MigrationsZip: path.Join(s.config.RootDir, "dist", "sk-migrations.zip"),
		ClientZip:     path.Join(s.config.RootDir, "dist", "sk-client.zip"),
		AppID:         2501,
		EnvID:         51191,
		DeploymentID:  202521,
	})

	s.NoError(err)
	s.Equal(int64(4102), result.Client.BytesUploaded)
	s.Equal(int64(15), result.Client.FilesUploaded)
	s.Equal(int64(0), result.Server.BytesUploaded)
	s.Equal(int64(1), result.Migrations.FilesUploaded)
	s.Equal(int64(512), result.Migrations.BytesUploaded)
}

func (s *UploaderSuite) Test_UploadLimits_StormkitCloud() {
	config.SetIsStormkitCloud(true)

	runner.Stat = func(name string) (os.FileInfo, error) {
		return MockFileInfo{
			size: 51 << 20, // 51 MB is not allowed so should return false
		}, nil
	}

	result, err := runner.NewUploader(s.config.Uploader).Upload(runner.UploadArgs{
		ClientZip: path.Join(s.config.RootDir, "dist", "sk-client.zip"),
	})

	msg := "Deployment size is larger than allowed.\nFor client-side applications, the limit is 50MB and for serverless applications 100MB."

	s.Error(err)
	s.Equal(msg, err.Error())
	s.Nil(result)
}

func (s *UploaderSuite) Test_NoUploadLimits_SelfHosted() {
	config.SetIsStormkitCloud(false)
	config.SetIsSelfHosted(true)

	s.mockIntegrations.On("Upload", integrations.UploadArgs{
		ClientZip: path.Join(s.config.RootDir, "dist", "sk-client.zip"),
	}).Return(&integrations.UploadResult{
		Client: integrations.UploadOverview{
			FilesUploaded: 15,
			BytesUploaded: 4102,
		},
	}, nil)

	runner.Stat = func(name string) (os.FileInfo, error) {
		return MockFileInfo{
			size: 51 << 20, // 51 MB is not allowed so should return false
		}, nil
	}

	result, err := runner.NewUploader(s.config.Uploader).Upload(runner.UploadArgs{
		ClientZip: path.Join(s.config.RootDir, "dist", "sk-client.zip"),
	})

	s.NoError(err)
	s.NotNil(result)
}

func TestUploaderSuite(t *testing.T) {
	suite.Run(t, &UploaderSuite{})
}
