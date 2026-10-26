// Package provisioners holds the provider-neutral parts of creating dedicated
// Stormkit Cloud instances: the instance model, input validation, the owner
// API key, the first-boot install script and the readiness check. Each
// hosting provider lives in its own subpackage (e.g. provisioners/hetzner)
// and implements Provisioner.
package provisioners

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
)

var (
	// ErrNoCapacity is returned when the provider cannot create a server with
	// any of the configured sizes in any of the configured locations.
	ErrNoCapacity = errors.New("no configured server type is in stock in any configured location")

	// ErrNotReady is returned when an instance does not accept its API key in
	// time. Providers delete the server before returning it.
	ErrNotReady = errors.New("instance did not become ready in time")
)

var (
	namePattern   = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,40}[a-z0-9])?$`)
	domainPattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$`)
)

// Provisioner creates and deletes Stormkit instances on one hosting provider.
type Provisioner interface {
	// CreateInstance creates a server, installs Stormkit on it and returns
	// once the instance accepts its API key.
	CreateInstance(ctx context.Context, params CreateInstanceParams) (*Instance, error)

	// DeleteInstance deletes the server backing an instance. Deleting a server
	// that no longer exists is not an error.
	DeleteInstance(ctx context.Context, serverID int64) error
}

// CreateInstanceParams describes the instance to create.
type CreateInstanceParams struct {
	// Name identifies the instance. Lowercase letters, digits and dashes.
	Name string

	// AdminEmail is the owner admin's email. Defaults to agent@<domain>.
	AdminEmail string

	// Domain is the instance's base domain. Defaults to <ip>.sslip.io.
	Domain string
}

// Validate reports every invalid field at once.
func (p CreateInstanceParams) Validate() error {
	errs := []error{}

	if !namePattern.MatchString(p.Name) {
		errs = append(errs, fmt.Errorf("name %q must be 1-42 lowercase letters, digits or dashes, not starting or ending with a dash", p.Name))
	}

	if p.AdminEmail != "" {
		addr, err := mail.ParseAddress(p.AdminEmail)

		if err != nil || addr.Address != p.AdminEmail {
			errs = append(errs, fmt.Errorf("admin email %q is not a plain email address", p.AdminEmail))
		}
	}

	if p.Domain != "" && (len(p.Domain) > 253 || !domainPattern.MatchString(p.Domain)) {
		errs = append(errs, fmt.Errorf("domain %q is not a valid hostname", p.Domain))
	}

	return errors.Join(errs...)
}

// Instance is a created and reachable Stormkit instance.
type Instance struct {
	Provider   string `json:"provider"`
	ServerID   int64  `json:"serverId"`
	Name       string `json:"name"`
	ServerType string `json:"serverType"`
	Location   string `json:"location"`
	IPv4       string `json:"ipv4"`
	IPv6       string `json:"ipv6"`
	Domain     string `json:"domain"`
	AdminEmail string `json:"adminEmail,omitempty"`

	// APIKey is the owner API key. The instance only stores its hash, so this
	// is the one place the raw value exists.
	APIKey string `json:"apiKey"`
}

// NewAPIKey generates an owner key in the same format install.sh mints.
func NewAPIKey() (string, error) {
	b := make([]byte, 31)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return "SK_" + hex.EncodeToString(b), nil
}
