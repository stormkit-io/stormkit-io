package provisioners

import (
	"strings"

	"github.com/goccy/go-yaml"
)

// installDir is where install.sh writes docker-compose.yaml and .env on the
// server. cloud-init runs commands from /, so the directory must be explicit.
const installDir = "/opt/stormkit"

// CloudInit renders the cloud-config a new server runs on first boot: it
// installs Stormkit in agent mode with a key the provisioner already holds.
type CloudInit struct {
	InstallScriptURL string
	APIKey           string
	AdminEmail       string
	Domain           string
}

// Render returns the cloud-config document.
func (c CloudInit) Render() (string, error) {
	args := []string{"--agent", "--api-key", c.quote(c.APIKey)}

	if c.AdminEmail != "" {
		args = append(args, "--email", c.quote(c.AdminEmail))
	}

	if c.Domain != "" {
		args = append(args, "--domain", c.quote(c.Domain))
	}

	install := "cd " + installDir +
		" && curl -fsSL " + c.quote(c.InstallScriptURL) +
		" | sh -s -- " + strings.Join(args, " ")

	out, err := yaml.Marshal(map[string]any{
		"runcmd": []string{
			"mkdir -p " + installDir,
			install,
		},
	})

	if err != nil {
		return "", err
	}

	return "#cloud-config\n" + string(out), nil
}

// quote wraps s in single quotes for sh, escaping embedded single quotes.
func (c CloudInit) quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
