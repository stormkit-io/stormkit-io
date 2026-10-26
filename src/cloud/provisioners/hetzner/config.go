package hetzner

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultServerTypes      = "cx23,cx33"
	defaultLocations        = "nbg1,fsn1,hel1"
	defaultImage            = "ubuntu-24.04"
	defaultInstallScriptURL = "https://www.stormkit.io/install.sh"
	defaultReadyTimeout     = 20 * time.Minute
)

// Config holds the settings needed to create Stormkit instances on Hetzner
// Cloud.
type Config struct {
	// Token is the Hetzner Cloud API token.
	Token string

	// ServerTypes are tried in order; the first one in stock wins.
	ServerTypes []string

	// Locations are tried in order for each server type.
	Locations []string

	// Image is the OS image new servers boot from.
	Image string

	// FirewallID attaches an existing Hetzner firewall when non-zero.
	FirewallID int64

	// SSHKeyIDs are existing Hetzner SSH keys that get root access, for
	// operators. At least one is required.
	SSHKeyIDs []int64

	// InstallScriptURL is where a new server downloads install.sh from.
	InstallScriptURL string

	// ReadyTimeout caps how long CreateInstance waits for the instance to answer.
	ReadyTimeout time.Duration
}

// ConfigFromEnv reads the provisioner configuration from environment variables.
func ConfigFromEnv() (Config, error) {
	r := &envReader{}

	cfg := Config{
		Token:            os.Getenv("HETZNER_API_KEY"),
		ServerTypes:      r.list(envVar{name: "HETZNER_SERVER_TYPES", fallback: defaultServerTypes}),
		Locations:        r.list(envVar{name: "HETZNER_LOCATIONS", fallback: defaultLocations}),
		Image:            r.str(envVar{name: "HETZNER_IMAGE", fallback: defaultImage}),
		FirewallID:       r.int64("HETZNER_FIREWALL_ID"),
		SSHKeyIDs:        r.int64List("HETZNER_SSH_KEY_IDS"),
		InstallScriptURL: r.str(envVar{name: "STORMKIT_INSTALL_SCRIPT_URL", fallback: defaultInstallScriptURL}),
		ReadyTimeout:     r.duration(durationVar{name: "PROVISIONER_READY_TIMEOUT", fallback: defaultReadyTimeout}),
	}

	if cfg.Token == "" {
		r.errs = append(r.errs, errors.New("HETZNER_API_KEY is required"))
	}

	if len(cfg.ServerTypes) == 0 {
		r.errs = append(r.errs, errors.New("HETZNER_SERVER_TYPES must list at least one server type"))
	}

	if len(cfg.Locations) == 0 {
		r.errs = append(r.errs, errors.New("HETZNER_LOCATIONS must list at least one location"))
	}

	// Without an SSH key Hetzner enables root password login and emails the
	// password for every server it creates.
	if len(cfg.SSHKeyIDs) == 0 {
		r.errs = append(r.errs, errors.New("HETZNER_SSH_KEY_IDS must list at least one SSH key"))
	}

	return cfg, errors.Join(r.errs...)
}

// envReader parses environment variables and collects every parse error, so
// a misconfigured deployment reports all its problems at once.
type envReader struct {
	errs []error
}

// envVar names a variable and the value to use when it is unset or blank.
type envVar struct {
	name     string
	fallback string
}

type durationVar struct {
	name     string
	fallback time.Duration
}

func (r *envReader) str(v envVar) string {
	if value := strings.TrimSpace(os.Getenv(v.name)); value != "" {
		return value
	}

	return v.fallback
}

func (r *envReader) list(v envVar) []string {
	items := []string{}

	for item := range strings.SplitSeq(r.str(v), ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}

	return items
}

func (r *envReader) int64(name string) int64 {
	v := r.str(envVar{name: name})

	if v == "" {
		return 0
	}

	n, err := strconv.ParseInt(v, 10, 64)

	if err != nil || n <= 0 {
		r.errs = append(r.errs, fmt.Errorf("%s must be a positive integer, got %q", name, v))
		return 0
	}

	return n
}

func (r *envReader) int64List(name string) []int64 {
	ids := []int64{}

	for _, item := range r.list(envVar{name: name}) {
		n, err := strconv.ParseInt(item, 10, 64)

		if err != nil || n <= 0 {
			r.errs = append(r.errs, fmt.Errorf("%s must be a comma-separated list of positive integers, got %q", name, item))
			continue
		}

		ids = append(ids, n)
	}

	return ids
}

func (r *envReader) duration(v durationVar) time.Duration {
	value := r.str(envVar{name: v.name})

	if value == "" {
		return v.fallback
	}

	d, err := time.ParseDuration(value)

	if err != nil || d <= 0 {
		r.errs = append(r.errs, fmt.Errorf("%s must be a positive duration like 20m, got %q", v.name, value))
		return v.fallback
	}

	return d
}
