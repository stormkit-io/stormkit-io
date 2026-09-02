package runner

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"github.com/stretchr/testify/suite"
)

type RepoSizerSuite struct {
	suite.Suite
	dir string
}

func (s *RepoSizerSuite) BeforeTest(_, _ string) {
	dir, err := os.MkdirTemp("", "tmp-test-repo-sizer-")

	s.NoError(err)
	s.dir = dir
}

func (s *RepoSizerSuite) AfterTest(_, _ string) {
	config.SetIsStormkitCloud(false)
	s.NoError(os.RemoveAll(s.dir))
}

func (s *RepoSizerSuite) git(args ...string) {
	cmd := sys.Command(context.Background(), sys.CommandOpts{
		Name: "git",
		Args: args,
		Dir:  s.dir,
		Env:  PrepareEnvVars(map[string]string{}),
	})

	out, err := cmd.CombinedOutput()
	s.Require().NoError(err, "git %v: %s", args, out)
}

// seedRepo builds a real repository with `count` files of `size` bytes each,
// then empties the working tree so the state matches a --no-checkout clone.
func (s *RepoSizerSuite) seedRepo(count, size int) {
	s.git("init", "-q", "-b", "main")
	s.git("config", "user.email", "test@example.com")
	s.git("config", "user.name", "Test")

	for i := range count {
		name := fmt.Sprintf("file-%d.bin", i)
		s.Require().NoError(os.WriteFile(path.Join(s.dir, name), make([]byte, size), 0644))
	}

	s.git("add", "-A")
	s.git("commit", "-qm", "seed")

	for i := range count {
		s.Require().NoError(os.Remove(path.Join(s.dir, fmt.Sprintf("file-%d.bin", i))))
	}
}

func (s *RepoSizerSuite) Test_WorkTreeSize_IsExactBeforeTheFilesExist() {
	s.seedRepo(4, 2048)

	sizer := repoSizer{dir: s.dir, vars: map[string]string{}}
	size, err := sizer.workTreeSize(context.Background())

	s.NoError(err)
	s.Equal(int64(4*2048), size, "measured from the objects, with no files on disk")
}

// The case RLIMIT_FSIZE cannot catch: lots of files, none of them large.
func (s *RepoSizerSuite) Test_WorkTreeSize_CountsManySmallFiles() {
	s.seedRepo(200, 4096)

	sizer := repoSizer{dir: s.dir, vars: map[string]string{}}
	size, err := sizer.workTreeSize(context.Background())

	s.NoError(err)
	s.Equal(int64(200*4096), size)
}

func (s *RepoSizerSuite) Test_WorkTreeSize_MatchesWhatCheckoutActuallyWrites() {
	s.seedRepo(12, 3000)

	sizer := repoSizer{dir: s.dir, vars: map[string]string{}}
	predicted, err := sizer.workTreeSize(context.Background())

	s.Require().NoError(err)

	s.git("checkout", "-f", "HEAD")

	var actual int64

	entries, err := os.ReadDir(s.dir)
	s.Require().NoError(err)

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		info, err := e.Info()
		s.Require().NoError(err)
		actual += info.Size()
	}

	s.Equal(actual, predicted, "the prediction must match the bytes checkout writes")
}

func (s *RepoSizerSuite) Test_Total_IncludesTheGitDirectory() {
	s.seedRepo(4, 2048)

	sizer := repoSizer{dir: s.dir, vars: map[string]string{}}
	total, err := sizer.total(context.Background())

	s.Require().NoError(err)
	s.Greater(total, int64(4*2048), ".git has to count toward the cap, it is on the same disk")
}

func (s *RepoSizerSuite) Test_WorkTreeSize_ErrorsOutsideARepository() {
	sizer := repoSizer{dir: s.dir, vars: map[string]string{}}
	_, err := sizer.workTreeSize(context.Background())

	s.Error(err, "a measurement failure must not be silently read as zero")
}

func (s *RepoSizerSuite) Test_MaxRepoSize_OnlyAppliesToCloud() {
	config.SetIsStormkitCloud(true)
	s.Equal(CloudMaxRepoSize, maxRepoSize())

	config.SetIsStormkitCloud(false)
	s.Equal(int64(0), maxRepoSize())
}

func (s *RepoSizerSuite) Test_MaxRepoSize_HonoursTheEnvOverride() {
	config.SetIsStormkitCloud(true)

	s.T().Setenv(MaxRepoSizeEnvVar, "250")
	s.Equal(int64(250<<20), maxRepoSize())
}

func (s *RepoSizerSuite) Test_MaxRepoSize_FallsBackOnAnUnusableOverride() {
	config.SetIsStormkitCloud(true)

	for _, bad := range []string{"not-a-number", "0", "-5"} {
		s.T().Setenv(MaxRepoSizeEnvVar, bad)
		s.Equal(CloudMaxRepoSize, maxRepoSize(), "override %q must not disable the cap", bad)
	}
}

func (s *RepoSizerSuite) Test_ErrRepoTooLarge_QuotesTheMeasuredSize() {
	err := ErrRepoTooLarge{size: 9830 << 20, limit: CloudMaxRepoSize}

	s.Contains(err.Error(), "The repository is 9.6GB while the limit is 1.0GB")
}

func (s *RepoSizerSuite) Test_ErrRepoTooLarge_OmitsAnUnknownSize() {
	// The kernel stopped the download, so there is no measurement to quote.
	err := ErrRepoTooLarge{limit: CloudMaxRepoSize}

	s.Contains(err.Error(), "The download exceeded the 1.0GB limit")
	s.NotContains(err.Error(), "while the limit")
}

func (s *RepoSizerSuite) Test_HumanBytes_SwitchesToGigabytes() {
	s.Equal("512.0MB", humanBytes(512<<20))
	s.Equal("1.0GB", humanBytes(1<<30))
	s.Equal("2.5GB", humanBytes(2<<30+512<<20))
}

func Test_RepoSizerSuite(t *testing.T) {
	suite.Run(t, new(RepoSizerSuite))
}

// CloneStderrSuite covers attributing a failed clone to the size cap. The
// kernel kills git index-pack rather than the git clone we wait on, so the
// wait status carries no signal and git's own message is all that is left.
type CloneStderrSuite struct {
	suite.Suite
}

func (s *CloneStderrSuite) Test_RecognisesARLimitFailure() {
	// The exact message git emits when RLIMIT_FSIZE stops the pack, captured
	// from a real clone under `ulimit -f`.
	for _, out := range []string{
		"Cloning into './dst'...\nfatal: fetch-pack: invalid index-pack output\n",
		"fatal: write error: File too large\n",
		"File size limit exceeded\n",
	} {
		stderr := &cloneStderr{}
		_, err := stderr.Write([]byte(out))

		s.Require().NoError(err)
		s.True(stderr.sawFileSizeLimit(), "should recognise %q", out)
	}
}

func (s *CloneStderrSuite) Test_IgnoresAnOrdinaryFailure() {
	stderr := &cloneStderr{}
	_, err := stderr.Write([]byte("fatal: could not read Username for 'https://github.com'\n"))

	s.Require().NoError(err)
	s.False(stderr.sawFileSizeLimit(), "an auth failure must not be reported as a size breach")
}

func (s *CloneStderrSuite) Test_KeepsOnlyTheTail() {
	stderr := &cloneStderr{}

	// A clone emits megabytes of progress; only the tail may be retained.
	_, err := stderr.Write(make([]byte, 64<<10))
	s.Require().NoError(err)

	s.LessOrEqual(len(stderr.tail), cloneStderrTail)

	// The signature must still be found when it arrives after the flood.
	_, err = stderr.Write([]byte("fatal: fetch-pack: invalid index-pack output\n"))
	s.Require().NoError(err)
	s.True(stderr.sawFileSizeLimit())
}

func (s *CloneStderrSuite) Test_MatchesAcrossSeparateWrites() {
	stderr := &cloneStderr{}

	// git writes progress in chunks, so a signature can straddle two calls.
	_, err := stderr.Write([]byte("fatal: fetch-pack: invalid "))
	s.Require().NoError(err)
	_, err = stderr.Write([]byte("index-pack output\n"))
	s.Require().NoError(err)

	s.True(stderr.sawFileSizeLimit())
}

func Test_CloneStderrSuite(t *testing.T) {
	suite.Run(t, new(CloneStderrSuite))
}
