package buildconf_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type EnvModelSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *EnvModelSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *EnvModelSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *EnvModelSuite) Test_Config_Validation() {
	config := &buildconf.Env{}

	s.Equal([]string{
		"Branch is a required field",
		"Name is a required field",
	}, buildconf.Validate(config))

	config.Name = "Some invalid env"
	config.Branch = "Valid-Env-1015+=/z"

	s.Equal([]string{
		"Environment name can only contain alphanumeric characters and hypens",
	}, buildconf.Validate(config))

	config.Name = "Some-Valid-Env-Name"
	config.Branch = "I'm invalid"

	s.Equal([]string{
		"Branch name can only contain following characters: alphanumeric, -, +, /, ., and =",
	}, buildconf.Validate(config))

	config.Branch = "valid-branch"
	config.AutoDeployBranches = null.StringFrom("(invalid-regex")

	s.Equal([]string{
		"Auto deploy branches regex is invalid: error parsing regexp: missing closing ) in `(invalid-regex`",
	}, buildconf.Validate(config))
}

func TestEnvModelSuite(t *testing.T) {
	suite.Run(t, &EnvModelSuite{})
}
