// Package hetzner provisions dedicated Stormkit instances on Hetzner Cloud.
package hetzner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/cloud/provisioners"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// ProviderName identifies Hetzner in provisioners.Instance.Provider.
const ProviderName = "hetzner"

const (
	// instanceLabel tags every server the provisioner creates with the
	// instance name. DeleteInstance refuses servers without it.
	instanceLabel = "stormkit-instance"

	// attemptLabel tags a server with the create call that made it, so a
	// create whose outcome is unknown can find and remove exactly its server.
	attemptLabel = "stormkit-attempt"
)

// cleanupTimeout bounds deletes that run after the caller's context failed.
const cleanupTimeout = time.Minute

// Provisioner creates and deletes Stormkit instances on Hetzner Cloud.
type Provisioner struct {
	cfg       Config
	hcloud    *hcloud.Client
	readiness *provisioners.Readiness
}

var _ provisioners.Provisioner = (*Provisioner)(nil)

// New returns a provisioner that talks to the Hetzner Cloud API.
func New(cfg Config) *Provisioner {
	return &Provisioner{
		cfg:       cfg,
		hcloud:    hcloud.NewClient(hcloud.WithToken(cfg.Token), hcloud.WithApplication("stormkit-provisioner", "")),
		readiness: provisioners.NewReadiness(cfg.ReadyTimeout),
	}
}

// CreateInstance creates a server with the first in-stock server type and
// location, installs Stormkit on it and waits until the instance accepts its
// API key. If the instance never becomes ready, the server is deleted.
func (p *Provisioner) CreateInstance(ctx context.Context, params provisioners.CreateInstanceParams) (*provisioners.Instance, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	apiKey := apikey.GenerateTokenValue()

	userData, err := provisioners.CloudInit{
		InstallScriptURL: p.cfg.InstallScriptURL,
		APIKey:           apiKey,
		AdminEmail:       params.AdminEmail,
	}.Render()

	if err != nil {
		return nil, err
	}

	created, err := p.createServer(ctx, createServerParams{
		name:     params.Name,
		userData: userData,
	})

	if err != nil {
		return nil, err
	}

	instance := &provisioners.Instance{
		Provider:   ProviderName,
		ServerID:   created.server.ID,
		Name:       params.Name,
		ServerType: created.serverType,
		Location:   created.location,
		AdminEmail: params.AdminEmail,
		APIKey:     apiKey,
	}

	if ip := created.server.PublicNet.IPv6.IP; ip != nil {
		instance.IPv6 = ip.String()
	}

	readyErr := errors.New("server has no public IPv4 address")

	if ip := created.server.PublicNet.IPv4.IP; ip != nil {
		instance.IPv4 = ip.String()
		instance.Domain = strings.ReplaceAll(instance.IPv4, ".", "-") + ".sslip.io"

		// Mirrors install.sh, which names the owner agent@<domain> by default.
		if instance.AdminEmail == "" {
			instance.AdminEmail = "agent@" + instance.Domain
		}

		readyErr = p.readiness.Wait(ctx, instance)
	}

	if readyErr != nil {
		// The caller's context may be what failed, so clean up on a fresh one.
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
		defer cancel()

		if err := p.DeleteInstance(cleanupCtx, instance.ServerID); err != nil {
			return nil, errors.Join(readyErr, fmt.Errorf("delete server %d: %w", instance.ServerID, err))
		}

		return nil, readyErr
	}

	return instance, nil
}

// DeleteInstance deletes the server backing an instance. It refuses servers
// the provisioner did not create, so a wrong ID cannot take down unrelated
// machines in the same Hetzner project.
func (p *Provisioner) DeleteInstance(ctx context.Context, serverID int64) error {
	server, _, err := p.hcloud.Server.GetByID(ctx, serverID)

	if err != nil {
		return err
	}

	if server == nil {
		return nil
	}

	if server.Labels[instanceLabel] == "" {
		return fmt.Errorf("server %d (%s) has no %s label; refusing to delete a server the provisioner did not create", serverID, server.Name, instanceLabel)
	}

	_, _, err = p.hcloud.Server.DeleteWithResult(ctx, server)

	if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
		return nil
	}

	return err
}

type createServerParams struct {
	name     string
	userData string
}

type createdServer struct {
	server     *hcloud.Server
	serverType string
	location   string
}

// createServer walks the configured server types and locations in order and
// creates the server in the first one that is in stock. Only capacity errors
// move on to the next option; anything else (bad token, quota, name taken)
// would fail the same way everywhere, so it is returned immediately.
func (p *Provisioner) createServer(ctx context.Context, params createServerParams) (*createdServer, error) {
	cat, err := p.catalog(ctx)

	if err != nil {
		return nil, fmt.Errorf("list server availability: %w", err)
	}

	if err := cat.validate(p.cfg); err != nil {
		return nil, err
	}

	attempt := utils.RandomToken(16)

	for _, serverType := range p.cfg.ServerTypes {
		for _, location := range p.cfg.Locations {
			if !cat.stock[location][serverType] {
				continue
			}

			result, _, err := p.hcloud.Server.Create(ctx, p.serverCreateOpts(serverCreateOptsParams{
				name:       params.name,
				attempt:    attempt,
				userData:   params.userData,
				serverType: serverType,
				location:   location,
			}))

			if hcloud.IsError(err, hcloud.ErrorCodeResourceUnavailable, hcloud.ErrorCodePlacementError) {
				continue
			}

			if err == nil && result.Server != nil {
				return &createdServer{
					server:     result.Server,
					serverType: serverType,
					location:   location,
				}, nil
			}

			if err == nil {
				err = errors.New("hetzner returned no server")
			}

			err = fmt.Errorf("create %s server in %s: %w", serverType, location, err)

			// Hetzner may have created the server even though we never saw it
			// (a cancelled request, a dropped response). It would keep running
			// and billing with a valid owner key in its user-data.
			if cleanupErr := p.deleteAttempt(ctx, attempt); cleanupErr != nil {
				return nil, errors.Join(err, fmt.Errorf("clean up servers of failed create: %w", cleanupErr))
			}

			return nil, err
		}
	}

	return nil, provisioners.ErrNoCapacity
}

// deleteAttempt deletes every server labelled with the given create attempt.
func (p *Provisioner) deleteAttempt(ctx context.Context, attempt string) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()

	servers, err := p.hcloud.Server.AllWithOpts(ctx, hcloud.ServerListOpts{
		ListOpts: hcloud.ListOpts{LabelSelector: attemptLabel + "=" + attempt},
	})

	if err != nil {
		return err
	}

	errs := []error{}

	for _, server := range servers {
		if err := p.DeleteInstance(ctx, server.ID); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

type serverCreateOptsParams struct {
	name       string
	attempt    string
	userData   string
	serverType string
	location   string
}

func (p *Provisioner) serverCreateOpts(params serverCreateOptsParams) hcloud.ServerCreateOpts {
	opts := hcloud.ServerCreateOpts{
		Name:             "stormkit-" + params.name,
		ServerType:       &hcloud.ServerType{Name: params.serverType},
		Image:            &hcloud.Image{Name: p.cfg.Image},
		Location:         &hcloud.Location{Name: params.location},
		UserData:         params.userData,
		StartAfterCreate: hcloud.Ptr(true),
		Labels: map[string]string{
			instanceLabel: params.name,
			attemptLabel:  params.attempt,
		},
		PublicNet: &hcloud.ServerCreatePublicNet{
			EnableIPv4: true,
			EnableIPv6: true,
		},
	}

	if p.cfg.FirewallID != 0 {
		opts.Firewalls = []*hcloud.ServerCreateFirewall{{Firewall: hcloud.Firewall{ID: p.cfg.FirewallID}}}
	}

	for _, id := range p.cfg.SSHKeyIDs {
		opts.SSHKeys = append(opts.SSHKeys, &hcloud.SSHKey{ID: id})
	}

	return opts
}

// catalog is what Hetzner offers right now.
type catalog struct {
	// serverTypes holds every server type name Hetzner knows.
	serverTypes map[string]bool

	// stock maps a location name to the server type names in stock there.
	stock map[string]map[string]bool
}

// validate reports configured server types and locations Hetzner does not
// know, which would otherwise look like a permanent capacity shortage.
func (c *catalog) validate(cfg Config) error {
	errs := []error{}

	for _, serverType := range cfg.ServerTypes {
		if !c.serverTypes[serverType] {
			errs = append(errs, fmt.Errorf("HETZNER_SERVER_TYPES: unknown server type %q", serverType))
		}
	}

	for _, location := range cfg.Locations {
		if _, ok := c.stock[location]; !ok {
			errs = append(errs, fmt.Errorf("HETZNER_LOCATIONS: unknown location %q", location))
		}
	}

	return errors.Join(errs...)
}

func (p *Provisioner) catalog(ctx context.Context) (*catalog, error) {
	serverTypes, err := p.hcloud.ServerType.All(ctx)

	if err != nil {
		return nil, err
	}

	cat := &catalog{
		serverTypes: map[string]bool{},
		stock:       map[string]map[string]bool{},
	}

	names := map[int64]string{}

	for _, st := range serverTypes {
		names[st.ID] = st.Name
		cat.serverTypes[st.Name] = true
	}

	datacenters, err := p.hcloud.Datacenter.All(ctx)

	if err != nil {
		return nil, err
	}

	for _, dc := range datacenters {
		if dc.Location == nil {
			continue
		}

		if cat.stock[dc.Location.Name] == nil {
			cat.stock[dc.Location.Name] = map[string]bool{}
		}

		for _, st := range dc.ServerTypes.Available {
			if name := names[st.ID]; name != "" {
				cat.stock[dc.Location.Name][name] = true
			}
		}
	}

	return cat, nil
}
