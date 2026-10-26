package provisioners

import (
	"strings"

	"github.com/goccy/go-yaml"
)

// installDir is where install.sh writes docker-compose.yaml and .env on the
// server. cloud-init runs commands from /, so the directory must be explicit.
const installDir = "/opt/stormkit"

// metadataIP is the link-local metadata service most cloud providers,
// Hetzner included, serve the server's user-data from.
const metadataIP = "169.254.169.254"

// CloudInit renders the cloud-config a new server runs on first boot: it
// installs Stormkit in agent mode with a key the provisioner already holds.
type CloudInit struct {
	InstallScriptURL string
	APIKey           string
	AdminEmail       string
}

// Render returns the cloud-config document.
//
// The user-data carries the owner API key and stays readable from the
// metadata service for the server's lifetime. Builds and functions run in
// containers on the instance, so bootcmd (which runs on every boot, before
// Docker starts) drops container traffic to the metadata service. Docker
// keeps rules it finds in DOCKER-USER, so creating the chain early is enough.
func (c CloudInit) Render() (string, error) {
	args := []string{"--agent", "--api-key", c.quote(c.APIKey)}

	if c.AdminEmail != "" {
		args = append(args, "--email", c.quote(c.AdminEmail))
	}

	install := "cd " + installDir +
		" && curl -fsSL " + c.quote(c.InstallScriptURL) +
		" | sh -s -- " + strings.Join(args, " ")

	dropMetadata := "-d " + metadataIP + " -j DROP"

	out, err := yaml.Marshal(map[string]any{
		"bootcmd": []string{
			"iptables -N DOCKER-USER 2>/dev/null || true",
			"iptables -C DOCKER-USER " + dropMetadata + " 2>/dev/null || iptables -I DOCKER-USER " + dropMetadata,
		},
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
