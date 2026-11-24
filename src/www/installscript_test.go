package www_test

import (
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

// InstallScriptSuite covers the install.sh flags Stormkit Cloud relies on.
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
	return s.runWithEnv(runWithEnvParams{args: args})
}

type runWithEnvParams struct {
	env  []string
	args []string
}

// runWithEnv is run with extra environment variables.
func (s *InstallScriptSuite) runWithEnv(p runWithEnvParams) (string, int) {
	cmd := exec.Command("sh", append([]string{s.script}, p.args...)...)
	cmd.Env = append(os.Environ(), p.env...)

	out, err := cmd.CombinedOutput()

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

func (s *InstallScriptSuite) Test_TrustedLoginURL_RequiresAgent() {
	out, code := s.run("--trusted-login-url", "https://console.stormkit.io/redeem")

	s.Equal(1, code)
	s.Contains(out, "--trusted-login-url requires --agent")
}

func (s *InstallScriptSuite) Test_TrustedLoginURL_RejectsBadFormat() {
	for _, u := range []string{"http://console.stormkit.io", "https://console.stormkit.io/?x=1", "https://console.stormkit.io/\"; rm -rf /", "https://a.io/$(id)"} {
		out, code := s.run("--agent", "--trusted-login-url="+u)

		s.Equal(1, code, u)
		s.Contains(out, "Invalid --trusted-login-url", u)
	}
}

func (s *InstallScriptSuite) Test_ConsoleCredential_RequiresAgent() {
	out, code := s.run("--console-credential", strings.Repeat("a", 43))

	s.Equal(1, code)
	s.Contains(out, "--console-credential requires --agent")
}

func (s *InstallScriptSuite) Test_ConsoleCredential_RejectsBadFormat() {
	for _, c := range []string{"short", strings.Repeat("a", 129), strings.Repeat("a", 40) + "/x", strings.Repeat("a", 40) + "\"; rm -rf /", strings.Repeat("a", 40) + "$(id)"} {
		out, code := s.run("--agent", "--console-credential="+c)

		s.Equal(1, code, c)
		s.Contains(out, "Invalid --console-credential", c)
	}
}

func (s *InstallScriptSuite) Test_AdminPassword_RequiresAgent() {
	out, code := s.run("--admin-password", strings.Repeat("a", 24))

	s.Equal(1, code)
	s.Contains(out, "--admin-password requires --agent")
}

func (s *InstallScriptSuite) Test_AdminPassword_RejectsBadFormat() {
	for _, p := range []string{"short", strings.Repeat("a", 129), strings.Repeat("a", 20) + "/x", strings.Repeat("a", 20) + "\"; rm -rf /", strings.Repeat("a", 20) + "$(id)"} {
		out, code := s.run("--agent", "--admin-password="+p)

		s.Equal(1, code, p)
		s.Contains(out, "Invalid --admin-password", p)
	}
}

type bootstrapServerParams struct {
	status int
	body   string
}

// bootstrapServer serves body with status to POSTs carrying bootstrapToken,
// and runs install.sh in bootstrap mode against it.
func (s *InstallScriptSuite) bootstrapServer(p bootstrapServerParams) (string, int) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer "+bootstrapToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(p.status)
		_, _ = w.Write([]byte(p.body))
	}))

	s.T().Cleanup(srv.Close)

	ca := filepath.Join(s.T().TempDir(), "ca.pem")
	cert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})

	s.Require().NoError(os.WriteFile(ca, cert, 0o600))

	return s.runWithEnv(runWithEnvParams{
		env:  []string{"CURL_CA_BUNDLE=" + ca},
		args: []string{"--agent", "--bootstrap-url", srv.URL + "/v1/instance/bootstrap", "--bootstrap-token", bootstrapToken},
	})
}

var bootstrapToken = strings.Repeat("t", 43)

func (s *InstallScriptSuite) Test_BootstrapURL_RequiresAgent() {
	out, code := s.run("--bootstrap-url", "https://console.stormkit.io/v1/instance/bootstrap", "--bootstrap-token", bootstrapToken)

	s.Equal(1, code)
	s.Contains(out, "--bootstrap-url requires --agent")
}

func (s *InstallScriptSuite) Test_BootstrapURL_RequiresToken() {
	out, code := s.run("--agent", "--bootstrap-url", "https://console.stormkit.io/v1/instance/bootstrap")

	s.Equal(1, code)
	s.Contains(out, "--bootstrap-url and --bootstrap-token go together")
}

func (s *InstallScriptSuite) Test_BootstrapURL_ReplacesSecretFlags() {
	for _, flag := range []string{"--api-key=SK_" + strings.Repeat("a", 62), "--admin-password=" + strings.Repeat("a", 24), "--console-credential=" + strings.Repeat("a", 43)} {
		out, code := s.run("--agent", "--bootstrap-url", "https://console.stormkit.io/v1/instance/bootstrap", "--bootstrap-token", bootstrapToken, flag)

		s.Equal(1, code, flag)
		s.Contains(out, "--bootstrap-url replaces", flag)
	}
}

func (s *InstallScriptSuite) Test_BootstrapURL_RejectsBadFormat() {
	for _, u := range []string{"http://console.stormkit.io/b", "https://console.stormkit.io/b?x=1", "https://a.io/$(id)"} {
		out, code := s.run("--agent", "--bootstrap-url="+u, "--bootstrap-token="+bootstrapToken)

		s.Equal(1, code, u)
		s.Contains(out, "Invalid --bootstrap-url", u)
	}

	for _, t := range []string{"short", strings.Repeat("a", 40) + "/x", strings.Repeat("a", 40) + "$(id)"} {
		out, code := s.run("--agent", "--bootstrap-url=https://console.stormkit.io/b", "--bootstrap-token="+t)

		s.Equal(1, code, t)
		s.Contains(out, "Invalid --bootstrap-token", t)
	}
}

// Test_BootstrapURL_Rejected verifies that a token the console refuses, such
// as one whose setup ended, stops the install at once.
func (s *InstallScriptSuite) Test_BootstrapURL_Rejected() {
	out, code := s.bootstrapServer(bootstrapServerParams{status: http.StatusUnauthorized})

	s.Equal(1, code)
	s.Contains(out, "Could not fetch the bootstrap secrets")
}

func (s *InstallScriptSuite) Test_BootstrapURL_MissingValues() {
	out, code := s.bootstrapServer(bootstrapServerParams{status: http.StatusOK, body: "api_key=SK_" + strings.Repeat("a", 62) + "\n"})

	s.Equal(1, code)
	s.Contains(out, "missing api_key or admin_password")
}

// Test_BootstrapURL_ChecksValues verifies that fetched secrets go through the
// same checks as the flags they replace, since they are written into .env.
func (s *InstallScriptSuite) Test_BootstrapURL_ChecksValues() {
	good := map[string]string{
		"api_key":            "SK_" + strings.Repeat("a", 62),
		"admin_password":     strings.Repeat("p", 24),
		"console_credential": strings.Repeat("c", 43),
	}

	cases := map[string]string{
		"api_key":            "Invalid --api-key",
		"admin_password":     "Invalid --admin-password",
		"console_credential": "Invalid --console-credential",
	}

	for bad, want := range cases {
		body := ""

		for name, value := range good {
			if name == bad {
				value += "\"; rm -rf /"
			}

			body += name + "=" + value + "\n"
		}

		out, code := s.bootstrapServer(bootstrapServerParams{status: http.StatusOK, body: body})

		s.Equal(1, code, bad)
		s.Contains(out, want, bad)
	}
}

func TestInstallScriptSuite(t *testing.T) {
	suite.Run(t, new(InstallScriptSuite))
}
