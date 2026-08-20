package buildconf_test

import (
	"encoding/json"
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
		"Environment name can only contain alphanumeric characters and hyphens",
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

func (s *EnvModelSuite) Test_JSON_MasksEnvVars() {
	env := buildconf.Env{
		Name:   "production",
		Branch: "main",
		Data: &buildconf.BuildConf{
			Vars: map[string]string{
				"SECRET":    "shh",
				"SK_SECRET": "also-shh",
			},
		},
	}

	build := env.JSON()["build"].(*buildconf.BuildConf)

	// Every value is masked (keys kept), including names that start with SK_.
	s.Equal("", build.Vars["SECRET"])
	s.Equal("", build.Vars["SK_SECRET"])

	// The original Env keeps the real values for internal use.
	s.Equal("shh", env.Data.Vars["SECRET"])
}

// Test_JSON_MasksMailerPassword guards the wide surface: every environment
// serialization (dashboard env GET, public API env list, the list_environments
// MCP tool) goes through Env.JSON, so the SMTP credential must not be in it.
func (s *EnvModelSuite) Test_JSON_MasksMailerPassword() {
	env := buildconf.Env{
		Name: "production",
		MailerConf: &buildconf.MailerConf{
			Host:     "smtp.gmail.com",
			Port:     "587",
			Username: "test",
			Password: "super-secret",
		},
	}

	mailer := env.JSON()["mailer"].(map[string]any)

	s.Equal(buildconf.PasswordPlaceholder, mailer["password"])
	s.Equal("smtp.gmail.com", mailer["host"])

	serialized, err := json.Marshal(env)
	s.NoError(err)
	s.NotContains(string(serialized), "super-secret")

	// The original Env keeps the real value for internal use.
	s.Equal("super-secret", env.MailerConf.Password)
}

func TestEnvModelSuite(t *testing.T) {
	suite.Run(t, &EnvModelSuite{})
}
