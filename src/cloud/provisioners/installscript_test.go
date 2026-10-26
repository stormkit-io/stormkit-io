package provisioners_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

// InstallScriptSuite covers the install.sh flags the provisioner relies on.
// Only argument checks that exit before anything is installed are exercised.
type InstallScriptSuite struct {
	suite.Suite

	script string
}

func (s *InstallScriptSuite) SetupSuite() {
	gomod, err := exec.Command("go", "env", "GOMOD").Output()
	s.Require().NoError(err)

	s.script = filepath.Join(filepath.Dir(strings.TrimSpace(string(gomod))), "src", "www", "public", "install.sh")
}

// run executes install.sh with args and returns its combined output and exit code.
func (s *InstallScriptSuite) run(args ...string) (string, int) {
	out, err := exec.Command("sh", append([]string{s.script}, args...)...).CombinedOutput()

	if exitErr, ok := err.(*exec.ExitError); ok {
		return string(out), exitErr.ExitCode()
	}

	s.Require().NoError(err)

	return string(out), 0
}

func (s *InstallScriptSuite) Test_Syntax() {
	out, err := exec.Command("sh", "-n", s.script).CombinedOutput()
	s.NoError(err, string(out))
}

func (s *InstallScriptSuite) Test_APIKey_RequiresAgent() {
	out, code := s.run("--api-key", "SK_"+strings.Repeat("a", 62))

	s.Equal(1, code)
	s.Contains(out, "--api-key requires --agent")
}

func (s *InstallScriptSuite) Test_APIKey_RejectsBadFormat() {
	for _, key := range []string{"abc", "SK_short", "SK_" + strings.Repeat("a", 31) + "/", "SK_" + strings.Repeat("a", 40) + "\"; rm -rf /"} {
		out, code := s.run("--agent", "--api-key="+key)

		s.Equal(1, code, key)
		s.Contains(out, "Invalid --api-key", key)
	}
}

func TestInstallScriptSuite(t *testing.T) {
	suite.Run(t, new(InstallScriptSuite))
}
