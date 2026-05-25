package runner_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	"github.com/stormkit-io/stormkit-io/src/ce/runner"
	"github.com/stretchr/testify/suite"
)

type NormalizeSuite struct {
	suite.Suite
}

func (s *NormalizeSuite) Test_PrefersTypedWorkDir() {
	msg := &deployservice.DeploymentMessage{
		Build: deployservice.BuildConfig{
			WorkDir: "./typed",
			Vars: map[string]string{
				"SK_CWD": "./legacy",
				"OTHER":  "value",
			},
		},
	}

	out := runner.Normalize(msg)

	s.Equal("./typed", out.Build.WorkDir)
	s.NotContains(out.Build.Vars, "SK_CWD")
	s.Equal("value", out.Build.Vars["OTHER"])
}

func (s *NormalizeSuite) Test_FallsBackToSKCWD() {
	msg := &deployservice.DeploymentMessage{
		Build: deployservice.BuildConfig{
			Vars: map[string]string{
				"SK_CWD": "./legacy",
			},
		},
	}

	out := runner.Normalize(msg)

	s.Equal("./legacy", out.Build.WorkDir)
	s.NotContains(out.Build.Vars, "SK_CWD")
}

func (s *NormalizeSuite) Test_NoWorkDirNoSKCWD() {
	msg := &deployservice.DeploymentMessage{
		Build: deployservice.BuildConfig{
			Vars: map[string]string{"FOO": "bar"},
		},
	}

	out := runner.Normalize(msg)

	s.Equal("", out.Build.WorkDir)
	s.Equal("bar", out.Build.Vars["FOO"])
}

func TestNormalizeSuite(t *testing.T) {
	suite.Run(t, &NormalizeSuite{})
}
