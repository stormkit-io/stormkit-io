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

type StatusChecksSuite struct {
	suite.Suite

	config  runner.RunnerOpts
	mockCmd *mocks.CommandInterface
}

func (s *StatusChecksSuite) BeforeTest(_, _ string) {
	s.mockCmd = &mocks.CommandInterface{}
	sys.DefaultCommand = s.mockCmd

	tmpDir, err := os.MkdirTemp("", "tmp-test-status-checks-")
	s.NoError(err)

	s.config = runner.RunnerOpts{
		Reporter: runner.NewReporter("https://example.com"),
		RootDir:  tmpDir,
		WorkDir:  path.Join(tmpDir, "repo"),
		Build: runner.BuildOpts{
			EnvVars: map[string]string{
				"NODE_ENV": "production",
				"TEST_RUN": "true",
			},
		},
	}

	s.NoError(s.config.MkdirAll())
}

func (s *StatusChecksSuite) AfterTest(_, _ string) {
	if strings.Contains(s.config.RootDir, os.TempDir()) {
		s.config.RemoveAll()
	}

	sys.DefaultCommand = nil
}

func (s *StatusChecksSuite) Test_Run() {
	sc := runner.NewStatusChecks(s.config)
	ctx := context.Background()

	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Name:   "sh",
		Args:   []string{"-c", "printenv"},
		Env:    runner.PrepareEnvVars(s.config.Build.EnvVars),
		Dir:    s.config.WorkDir,
		Stdout: s.config.Reporter.File(),
		Stderr: s.config.Reporter.File(),
	}).Return(s.mockCmd)

	s.mockCmd.On("Run").Return(nil)

	s.NoError(sc.Run(ctx, "printenv"))
}

func TestStatusChecksSuite(t *testing.T) {
	suite.Run(t, &StatusChecksSuite{})
}
