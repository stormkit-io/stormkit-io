package runner_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/runner"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type BuildManagerSuite struct {
	suite.Suite
	config runner.RunnerOpts
}

func (s *BuildManagerSuite) BeforeTest(_, _ string) {
	tmpDir, err := os.MkdirTemp("", "tmp-test-runner-")

	s.NoError(err)

	s.config = runner.RunnerOpts{
		RootDir:     tmpDir,
		Reporter:    runner.NewReporter("https://example.com"),
		AutoInstall: true,
		Repo: runner.RepoOpts{
			Dir: path.Join(tmpDir, "repo"),
		},
	}

	s.NoError(s.config.MkdirAll())
}

func (s *BuildManagerSuite) AfterTest(_, _ string) {
	if strings.Contains(s.config.RootDir, os.TempDir()) {
		s.config.RemoveAll()
	}

	s.config.Reporter.Close(nil, nil, nil)
}

func (s *BuildManagerSuite) Test_Build_Chaining() {
	s.config.Build.BuildCmd = "echo 'hello world' && echo 'hi world'"

	bm := runner.NewBuilder(s.config)
	s.NoError(bm.ExecCommands(context.Background()))

	lines := []string{
		fmt.Sprintf(`"title":%q`, s.config.Build.BuildCmd),
		"hello world",
		"hi world",
	}

	logs := s.config.Reporter.Logs()

	for _, line := range lines {
		s.Contains(logs, line)
	}
}

func (s *BuildManagerSuite) Test_Build_StaticRepo() {
	bm := runner.NewBuilder(s.config)
	s.NoError(bm.ExecCommands(context.Background()))

	logs := s.config.Reporter.Logs()
	s.Equal("", logs)
}

func (s *BuildManagerSuite) Test_Build_WithFlake() {
	s.config.WorkDir = s.config.Repo.Dir
	s.config.Build.BuildCmd = "npm run build"
	s.config.Build.EnvVars = map[string]string{}

	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "flake.nix"), []byte("{}"), 0776))

	mockCmd := &mocks.CommandInterface{}
	sys.DefaultCommand = mockCmd
	defer func() { sys.DefaultCommand = nil }()

	mockCmd.On("SetOpts", sys.CommandOpts{
		Name:   "nix",
		Args:   []string{"--extra-experimental-features", "nix-command flakes", "develop", "--command", "sh", "-c", "npm run build"},
		Env:    runner.PrepareEnvVars(s.config.Build.EnvVars),
		Dir:    s.config.Repo.Dir,
		Stdout: s.config.Reporter.File(),
		Stderr: s.config.Reporter.File(),
	}).Return(mockCmd).Once()

	mockCmd.On("Run").Return(nil, nil).Once()

	bm := runner.NewBuilder(s.config)
	s.NoError(bm.ExecCommands(context.Background()))
	s.Contains(s.config.Reporter.Logs(), `"title":"npm run build"`)
	mockCmd.AssertExpectations(s.T())
}

func (s *BuildManagerSuite) Test_Build_WithFlake_InRepoRoot() {
	subDir := path.Join(s.config.Repo.Dir, "subdir")
	s.NoError(os.MkdirAll(subDir, 0776))

	s.config.WorkDir = subDir
	s.config.Build.BuildCmd = "npm run build"
	s.config.Build.EnvVars = map[string]string{}

	// flake.nix is at repo root, not workDir
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "flake.nix"), []byte("{}"), 0776))

	mockCmd := &mocks.CommandInterface{}
	sys.DefaultCommand = mockCmd
	defer func() { sys.DefaultCommand = nil }()

	mockCmd.On("SetOpts", sys.CommandOpts{
		Name:   "nix",
		Args:   []string{"--extra-experimental-features", "nix-command flakes", "develop", "--command", "sh", "-c", "npm run build"},
		Env:    runner.PrepareEnvVars(s.config.Build.EnvVars),
		Dir:    s.config.Repo.Dir,
		Stdout: s.config.Reporter.File(),
		Stderr: s.config.Reporter.File(),
	}).Return(mockCmd).Once()

	mockCmd.On("Run").Return(nil, nil).Once()

	bm := runner.NewBuilder(s.config)
	s.NoError(bm.ExecCommands(context.Background()))
	mockCmd.AssertExpectations(s.T())
}

func TestBuildManagerSuite(t *testing.T) {
	suite.Run(t, &BuildManagerSuite{})
}
