package runner_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/runner"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"github.com/stretchr/testify/suite"
)

// RepoCheckoutSizeSuite drives Checkout against a real repository on disk, so
// the cap is exercised through actual git rather than a mocked command.
type RepoCheckoutSizeSuite struct {
	suite.Suite
	tmpDir string
	origin string
	config runner.RunnerOpts
}

func (s *RepoCheckoutSizeSuite) BeforeTest(_, _ string) {
	tmpDir, err := os.MkdirTemp("", "tmp-test-checkout-size-")

	s.NoError(err)

	s.tmpDir = tmpDir
	s.origin = path.Join(tmpDir, "origin")
	sys.DefaultCommand = nil
	runner.DefaultRepo = nil

	s.config = runner.RunnerOpts{
		RootDir:  tmpDir,
		Reporter: runner.NewReporter("https://example.com"),
		Build:    runner.BuildOpts{EnvVars: map[string]string{}},
		Repo: runner.RepoOpts{
			Dir:     path.Join(tmpDir, "repo"),
			Address: s.origin,
			Branch:  "main",
		},
	}

	s.NoError(s.config.MkdirAll())
	s.NoError(os.RemoveAll(s.config.Repo.Dir))
}

func (s *RepoCheckoutSizeSuite) AfterTest(_, _ string) {
	config.SetIsStormkitCloud(false)
	s.config.Reporter.Close(nil, nil, nil)
	runner.DefaultRepo = nil
	s.NoError(os.RemoveAll(s.tmpDir))
}

func (s *RepoCheckoutSizeSuite) git(dir string, args ...string) {
	cmd := sys.Command(context.Background(), sys.CommandOpts{
		Name: "git",
		Args: args,
		Dir:  dir,
		Env:  runner.PrepareEnvVars(map[string]string{}),
	})

	out, err := cmd.CombinedOutput()
	s.Require().NoError(err, "git %v: %s", args, out)
}

// seedOrigin creates a repository to clone from, holding `count` files of
// `size` bytes. Many small files is the shape a per-file limit cannot catch.
func (s *RepoCheckoutSizeSuite) seedOrigin(count, size int) {
	s.Require().NoError(os.MkdirAll(s.origin, 0755))

	s.git(s.origin, "init", "-q", "-b", "main")
	s.git(s.origin, "config", "user.email", "test@example.com")
	s.git(s.origin, "config", "user.name", "Test")

	for i := range count {
		name := path.Join(s.origin, fmt.Sprintf("file-%d.bin", i))
		s.Require().NoError(os.WriteFile(name, make([]byte, size), 0644))
	}

	s.git(s.origin, "add", "-A")
	s.git(s.origin, "commit", "-qm", "seed")
}

func (s *RepoCheckoutSizeSuite) workTreeFiles() int {
	entries, err := os.ReadDir(s.config.Repo.Dir)

	if err != nil {
		return 0
	}

	count := 0

	for _, e := range entries {
		if !e.IsDir() {
			count++
		}
	}

	return count
}

func (s *RepoCheckoutSizeSuite) Test_Checkout_PopulatesTheWorkingTreeUnderTheCap() {
	config.SetIsStormkitCloud(true)
	s.T().Setenv(runner.MaxRepoSizeEnvVar, "64")
	s.seedOrigin(5, 1024)

	s.NoError(runner.NewRepo(s.config).Checkout(context.Background()))
	s.Equal(5, s.workTreeFiles(), "an allowed repository must still be checked out")
}

// The case that motivated the change: many files, none of them individually
// large, adding up to more than the cap.
func (s *RepoCheckoutSizeSuite) Test_Checkout_RejectsManySmallFilesOverTheCap() {
	config.SetIsStormkitCloud(true)
	s.T().Setenv(runner.MaxRepoSizeEnvVar, "1")
	s.seedOrigin(400, 8192)

	err := runner.NewRepo(s.config).Checkout(context.Background())

	s.Require().Error(err)
	s.IsType(runner.ErrRepoTooLarge{}, err)
	s.Contains(err.Error(), "while the limit is 1.0MB")

	// The whole point: rejection happens before the working tree is written.
	s.Equal(0, s.workTreeFiles(), "no working-tree file may be written when over the cap")
}

func (s *RepoCheckoutSizeSuite) Test_Checkout_IsUncappedOutsideCloud() {
	config.SetIsStormkitCloud(false)
	s.T().Setenv(runner.MaxRepoSizeEnvVar, "1")
	s.seedOrigin(400, 8192)

	s.NoError(
		runner.NewRepo(s.config).Checkout(context.Background()),
		"self-hosted instances own their disk and must not be capped",
	)
	s.Equal(400, s.workTreeFiles())
}

func Test_RepoCheckoutSizeSuite(t *testing.T) {
	suite.Run(t, new(RepoCheckoutSizeSuite))
}
