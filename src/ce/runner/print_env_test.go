package runner

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type PrintEnvVariablesSuite struct {
	suite.Suite
}

func (s *PrintEnvVariablesSuite) Test_MasksAllValues() {
	reporter := NewReporter("")

	opts := RunnerOpts{
		Reporter: reporter,
		Build: BuildOpts{
			EnvVars: map[string]string{
				"NODE_ENV":     "production",
				"DATABASE_URL": "postgres://user:pass@host/db",
				"API_TOKEN":    "super-secret-token",
				"SK_ENV":       "production",
				"SK_APP_ID":    "8",
				// A user-defined SK_-prefixed variable must still be masked.
				"SK_MY_SECRET": "do-not-leak",
			},
		},
	}

	s.NoError(printEnvVariables(opts))

	log := reporter.Logs()

	// User-defined variable names are still listed so the build log stays
	// useful, but their values are masked.
	s.Contains(log, "NODE_ENV=***************")
	s.Contains(log, "DATABASE_URL=***************")
	s.Contains(log, "API_TOKEN=***************")
	s.NotContains(log, "postgres://user:pass@host/db")
	s.NotContains(log, "super-secret-token")

	// Only the allowlisted Stormkit system variables keep their values; a
	// user-defined SK_-prefixed variable is masked like any other secret.
	s.Contains(log, "SK_ENV=production")
	s.Contains(log, "SK_APP_ID=8")
	s.Contains(log, "SK_MY_SECRET=***************")
	s.NotContains(log, "do-not-leak")
}

func TestPrintEnvVariablesSuite(t *testing.T) {
	suite.Run(t, new(PrintEnvVariablesSuite))
}
