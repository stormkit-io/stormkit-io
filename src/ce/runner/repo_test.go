package runner_test

import (
	"context"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/runner"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type RepoSuite struct {
	suite.Suite
	tmpDir  string
	config  runner.RunnerOpts
	mockCmd *mocks.CommandInterface
}

func (s *RepoSuite) BeforeTest(_, _ string) {
	tmpDir, err := os.MkdirTemp("", "tmp-test-runner-")

	s.NoError(err)

	s.config = runner.RunnerOpts{
		RootDir:  tmpDir,
		Reporter: runner.NewReporter("https://example.com"),
		Build: runner.BuildOpts{
			EnvVars: map[string]string{},
		},
		Repo: runner.RepoOpts{
			Dir: path.Join(tmpDir, "repo"),
		},
	}

	s.NoError(s.config.MkdirAll())

	s.tmpDir = tmpDir
	s.mockCmd = &mocks.CommandInterface{}
	sys.DefaultCommand = s.mockCmd
}

func (s *RepoSuite) AfterTest(_, _ string) {
	if strings.Contains(s.config.RootDir, os.TempDir()) {
		s.config.RemoveAll()
	}

	s.config.Reporter.Close(nil, nil, nil)
	sys.DefaultCommand = nil
}

func (s *RepoSuite) Test_Constructor_Success() {
	var r runner.RepoInterface
	var err error

	opts := s.config
	opts.Repo.Address = "https://github.com/stormkit-io/stormkit-io"
	opts.Repo.AccessToken = "some-token"

	r = runner.NewRepo(opts)

	s.Nil(err)
	s.True(r.IsGithub())
	s.False(r.IsGitlab())
	s.False(r.IsBitbucket())
	s.Equal(r.Address(), "https://github.com/stormkit-io/stormkit-io")

	opts.Repo.Address = "https://gitlab.com/stormkit-io/stormkit-io"
	r = runner.NewRepo(opts)

	s.Nil(err)
	s.False(r.IsGithub())
	s.True(r.IsGitlab())
	s.False(r.IsBitbucket())
	s.Equal(r.Address(), "https://gitlab.com/stormkit-io/stormkit-io")

	opts.Repo.Address = "git@bitbucket.org/stormkit-io/stormkit-io"

	r = runner.NewRepo(opts)

	s.Nil(err)
	s.False(r.IsGithub())
	s.False(r.IsGitlab())
	s.True(r.IsBitbucket())
	s.Equal(r.Address(), "git@bitbucket.org/stormkit-io/stormkit-io")
}

func (s *RepoSuite) Test_Checkout_Github() {
	opts := s.config
	opts.Repo.Address = "https://github.com/stormkit-dev/e2e-npm"
	opts.Repo.AccessToken = "some-token"
	opts.Repo.Branch = "main"

	r := runner.NewRepo(opts)

	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Name: "git",
		Args: []string{
			"clone",
			"https://x-access-token:some-token@github.com/stormkit-dev/e2e-npm",
			"--depth", "1",
			"--progress",
			"--single-branch",
			"--branch", "main",
			s.config.Repo.Dir,
		},
		Env:    runner.PrepareEnvVars(map[string]string{"SK_BRANCH_NAME": "main"}),
		Dir:    s.config.WorkDir,
		Stderr: s.config.Reporter.File(),
		Stdout: s.config.Reporter.File(),
	}).Return(s.mockCmd)

	s.mockCmd.On("Run").Return(nil, nil)

	s.NoError(r.Checkout(context.Background()))
}

func (s *RepoSuite) Test_CommitInfo() {
	// Head SHA
	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Name: "git",
		Args: []string{
			"rev-parse",
			"HEAD",
		},
		Dir: s.config.Repo.Dir,
		Env:    runner.PrepareEnvVars(s.config.Build.EnvVars),
	}).Return(s.mockCmd).Once()

	s.mockCmd.On("Output").Return([]byte("790dcef2a8c61ff6011a4b595cdcb2f0de6c4e2b\n"), nil).Once()

	// Author Info
	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Name: "git",
		Args: []string{
			"--no-pager",
			"show",
			"-s",
			"--format='%an <%ae>'",
			"HEAD",
		},
		Dir: s.config.Repo.Dir,
		Env:    runner.PrepareEnvVars(s.config.Build.EnvVars),
	}).Return(s.mockCmd).Once()

	s.mockCmd.On("Output").Return([]byte("Joe Doe <joe@doe.org>\n"), nil).Once()

	// Message
	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Name: "git",
		Args: []string{"log", "-1", "--pretty=%B"},
		Dir:  s.config.Repo.Dir,
		Env:    runner.PrepareEnvVars(s.config.Build.EnvVars),
	}).Return(s.mockCmd).Once()

	s.mockCmd.On("Output").Return([]byte("chore: first commit\n\ncommit body"), nil).Once()

	r := runner.NewRepo(s.config)
	info := r.CommitInfo()

	s.Equal("790dcef2a8c61ff6011a4b595cdcb2f0de6c4e2b", info["sha"])
	s.Equal("Joe Doe <joe@doe.org>", info["author"])
	s.Equal("chore: first commit", info["message"])
}

func TestRepoSuite(t *testing.T) {
	suite.Run(t, &RepoSuite{})
}
