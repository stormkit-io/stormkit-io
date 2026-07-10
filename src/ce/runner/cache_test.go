package runner

import (
	"context"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"github.com/stretchr/testify/suite"
)

// mockCacheStore keeps cache archives in a local directory so Snapshot and
// Restore can round-trip without a real storage provider.
type mockCacheStore struct {
	storeDir  string
	uploads   int
	downloads int
}

func (m *mockCacheStore) archivePath() string {
	return path.Join(m.storeDir, "cache.tar.gz")
}

func (m *mockCacheStore) UploadCacheArtifact(_ context.Context, args integrations.CacheArtifactArgs) error {
	m.uploads++
	return file.Copy(args.LocalPath, m.archivePath(), 0664)
}

func (m *mockCacheStore) DownloadCacheArtifact(_ context.Context, args integrations.CacheArtifactArgs) (bool, error) {
	m.downloads++

	if _, err := os.Stat(m.archivePath()); err != nil {
		return false, nil
	}

	return true, file.Copy(m.archivePath(), args.LocalPath, 0664)
}

func (m *mockCacheStore) DeleteCacheArtifact(_ context.Context, _ integrations.CacheArtifactArgs) error {
	return os.Remove(m.archivePath())
}

type CacheManagerSuite struct {
	suite.Suite

	store *mockCacheStore
	opts  RunnerOpts
}

func (s *CacheManagerSuite) SetupTest() {
	rootDir := s.T().TempDir()
	workDir := path.Join(rootDir, "repo")

	s.Require().NoError(os.MkdirAll(workDir, 0775))

	s.store = &mockCacheStore{storeDir: s.T().TempDir()}
	DefaultCacheStore = s.store

	s.opts = RunnerOpts{
		RootDir:  rootDir,
		WorkDir:  workDir,
		Reporter: NewReporter(""),
		Build: BuildOpts{
			AppID:     "1",
			EnvID:     "2",
			CacheDirs: []string{"node_modules", ".next/cache"},
		},
	}
}

func (s *CacheManagerSuite) TearDownTest() {
	DefaultCacheStore = nil
}

func (s *CacheManagerSuite) writeWorkDirFile(relPath, content string) {
	fullPath := path.Join(s.opts.WorkDir, relPath)

	s.Require().NoError(os.MkdirAll(path.Dir(fullPath), 0775))
	s.Require().NoError(os.WriteFile(fullPath, []byte(content), 0664))
}

func (s *CacheManagerSuite) Test_SnapshotAndRestore_RoundTrip() {
	ctx := context.Background()

	s.writeWorkDirFile("node_modules/pkg/index.js", "module.exports = {}")
	s.writeWorkDirFile(".next/cache/build.json", "{}")
	s.writeWorkDirFile("src/app.js", "should not be cached")

	newCacheManager(s.opts).Snapshot(ctx)

	s.Equal(1, s.store.uploads)

	// Simulate a fresh checkout: wipe the working directory.
	s.Require().NoError(os.RemoveAll(s.opts.WorkDir))
	s.Require().NoError(os.MkdirAll(s.opts.WorkDir, 0775))

	newCacheManager(s.opts).Restore(ctx)

	content, err := os.ReadFile(path.Join(s.opts.WorkDir, "node_modules/pkg/index.js"))

	s.NoError(err)
	s.Equal("module.exports = {}", string(content))
	s.FileExists(path.Join(s.opts.WorkDir, ".next/cache/build.json"))
	s.NoFileExists(path.Join(s.opts.WorkDir, "src/app.js"))
}

func (s *CacheManagerSuite) Test_Restore_CacheMiss() {
	newCacheManager(s.opts).Restore(context.Background())

	s.Equal(1, s.store.downloads)

	entries, err := os.ReadDir(s.opts.WorkDir)

	s.NoError(err)
	s.Empty(entries)
}

// The deployer clears CacheDirs when caching is not allowed, so an empty
// list is how the runner sees a disabled cache.
func (s *CacheManagerSuite) Test_Disabled_WhenNoCacheDirs() {
	s.writeWorkDirFile("node_modules/pkg/index.js", "cached")
	s.opts.Build.CacheDirs = nil

	cache := newCacheManager(s.opts)
	cache.Snapshot(context.Background())
	cache.Restore(context.Background())

	s.Equal(0, s.store.uploads)
	s.Equal(0, s.store.downloads)
}

func (s *CacheManagerSuite) Test_Disabled_WhenNoDirs() {
	s.opts.Build.CacheDirs = nil

	newCacheManager(s.opts).Snapshot(context.Background())

	s.Equal(0, s.store.uploads)
}

func (s *CacheManagerSuite) Test_InvalidDirs_AreSkipped() {
	s.opts.Build.CacheDirs = []string{"../escape", "/absolute", "node_modules"}

	s.Equal([]string{"node_modules"}, newCacheManager(s.opts).dirs())
}

func (s *CacheManagerSuite) Test_Disabled_WhenAllDirsInvalid() {
	s.opts.Build.CacheDirs = []string{"../escape", "/absolute"}

	s.False(newCacheManager(s.opts).enabled())
}

func (s *CacheManagerSuite) Test_Snapshot_SkipsMissingDirs() {
	s.writeWorkDirFile("node_modules/pkg/index.js", "cached")

	// .next/cache does not exist; only node_modules should be archived.
	newCacheManager(s.opts).Snapshot(context.Background())

	s.Equal(1, s.store.uploads)

	s.Require().NoError(os.RemoveAll(s.opts.WorkDir))
	s.Require().NoError(os.MkdirAll(s.opts.WorkDir, 0775))

	newCacheManager(s.opts).Restore(context.Background())

	s.FileExists(path.Join(s.opts.WorkDir, "node_modules/pkg/index.js"))
	s.NoDirExists(path.Join(s.opts.WorkDir, ".next"))
}

func (s *CacheManagerSuite) Test_Snapshot_SkipsUpload_WhenUnchanged() {
	ctx := context.Background()

	s.writeWorkDirFile("node_modules/pkg/index.js", "module.exports = {}")

	newCacheManager(s.opts).Snapshot(ctx)

	s.Equal(1, s.store.uploads)

	s.Require().NoError(os.RemoveAll(s.opts.WorkDir))
	s.Require().NoError(os.MkdirAll(s.opts.WorkDir, 0775))

	cache := newCacheManager(s.opts)
	cache.Restore(ctx)

	// Package managers rewrite modification times on every install; only a
	// content change should trigger a re-upload.
	newTime := time.Now().Add(time.Hour)
	s.Require().NoError(os.Chtimes(path.Join(s.opts.WorkDir, "node_modules/pkg/index.js"), newTime, newTime))

	cache.Snapshot(ctx)

	s.Equal(1, s.store.uploads)
}

func (s *CacheManagerSuite) Test_Snapshot_Uploads_WhenContentChanged() {
	ctx := context.Background()

	s.writeWorkDirFile("node_modules/pkg/index.js", "module.exports = {}")

	newCacheManager(s.opts).Snapshot(ctx)

	s.Require().NoError(os.RemoveAll(s.opts.WorkDir))
	s.Require().NoError(os.MkdirAll(s.opts.WorkDir, 0775))

	cache := newCacheManager(s.opts)
	cache.Restore(ctx)

	// Same byte length as the original content: a size check would miss this.
	s.writeWorkDirFile("node_modules/pkg/index.js", "module.exports = []")

	cache.Snapshot(ctx)

	s.Equal(2, s.store.uploads)
}

func (s *CacheManagerSuite) Test_Snapshot_Uploads_WhenNewDirAppears() {
	ctx := context.Background()

	s.writeWorkDirFile("node_modules/pkg/index.js", "module.exports = {}")

	newCacheManager(s.opts).Snapshot(ctx)

	s.Require().NoError(os.RemoveAll(s.opts.WorkDir))
	s.Require().NoError(os.MkdirAll(s.opts.WorkDir, 0775))

	cache := newCacheManager(s.opts)
	cache.Restore(ctx)

	s.writeWorkDirFile(".next/cache/build.json", "{}")

	cache.Snapshot(ctx)

	s.Equal(2, s.store.uploads)
}

func (s *CacheManagerSuite) Test_Snapshot_Uploads_OnCacheMiss() {
	ctx := context.Background()

	cache := newCacheManager(s.opts)
	cache.Restore(ctx)

	s.writeWorkDirFile("node_modules/pkg/index.js", "module.exports = {}")

	cache.Snapshot(ctx)

	s.Equal(1, s.store.uploads)
}

func (s *CacheManagerSuite) Test_Snapshot_NoUpload_WhenNoDirsExist() {
	newCacheManager(s.opts).Snapshot(context.Background())

	s.Equal(0, s.store.uploads)
}

func TestCacheManagerSuite(t *testing.T) {
	suite.Run(t, new(CacheManagerSuite))
}
