package integrations_test

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stretchr/testify/suite"
)

type FilesysCacheStoreSuite struct {
	suite.Suite

	prevStorageDir string
	localDir       string
}

func (s *FilesysCacheStoreSuite) SetupTest() {
	s.prevStorageDir = config.Get().Deployer.StorageDir
	config.Get().Deployer.StorageDir = s.T().TempDir()
	s.localDir = s.T().TempDir()
}

func (s *FilesysCacheStoreSuite) TearDownTest() {
	config.Get().Deployer.StorageDir = s.prevStorageDir
}

func (s *FilesysCacheStoreSuite) args(localPath string) integrations.CacheArtifactArgs {
	return integrations.CacheArtifactArgs{
		AppID:     1,
		EnvID:     2,
		Dir:       "node_modules",
		LocalPath: localPath,
	}
}

func (s *FilesysCacheStoreSuite) Test_UploadDownload_RoundTrip() {
	ctx := context.Background()
	client := integrations.Filesys()

	archive := path.Join(s.localDir, "cache.tar.gz")
	s.Require().NoError(os.WriteFile(archive, []byte("archive-content"), 0664))

	s.NoError(client.UploadCacheArtifact(ctx, s.args(archive)))

	downloaded := path.Join(s.localDir, "downloaded.tar.gz")
	found, err := client.DownloadCacheArtifact(ctx, s.args(downloaded))

	s.NoError(err)
	s.True(found)

	content, err := os.ReadFile(downloaded)

	s.NoError(err)
	s.Equal("archive-content", string(content))
}

func (s *FilesysCacheStoreSuite) Test_Download_CacheMiss() {
	found, err := integrations.Filesys().DownloadCacheArtifact(
		context.Background(),
		s.args(path.Join(s.localDir, "missing.tar.gz")),
	)

	s.NoError(err)
	s.False(found)
}

func (s *FilesysCacheStoreSuite) Test_CachesAreScopedPerEnvironment() {
	ctx := context.Background()
	client := integrations.Filesys()

	archive := path.Join(s.localDir, "cache.tar.gz")
	s.Require().NoError(os.WriteFile(archive, []byte("env-2-cache"), 0664))

	s.NoError(client.UploadCacheArtifact(ctx, s.args(archive)))

	otherEnv := integrations.CacheArtifactArgs{
		AppID:     1,
		EnvID:     3,
		Dir:       "node_modules",
		LocalPath: path.Join(s.localDir, "other.tar.gz"),
	}

	found, err := client.DownloadCacheArtifact(ctx, otherEnv)

	s.NoError(err)
	s.False(found)
}

func TestFilesysCacheStoreSuite(t *testing.T) {
	suite.Run(t, new(FilesysCacheStoreSuite))
}
