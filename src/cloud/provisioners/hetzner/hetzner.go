// Package hetzner provisions dedicated Stormkit instances on Hetzner Cloud.
package hetzner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/stormkit-io/stormkit-io/src/cloud/provisioners"
)

// ProviderName identifies Hetzner in provisioners.Instance.Provider.
const ProviderName = "hetzner"

// instanceLabel tags every server the provisioner creates with the instance name.
const instanceLabel = "stormkit-instance"

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

	apiKey, err := provisioners.NewAPIKey()

	if err != nil {
		return nil, err
	}

	userData, err := provisioners.CloudInit{
		InstallScriptURL: p.cfg.InstallScriptURL,
		APIKey:           apiKey,
		AdminEmail:       params.AdminEmail,
		Domain:           params.Domain,
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
		Domain:     params.Domain,
		AdminEmail: params.AdminEmail,
		APIKey:     apiKey,
	}

	if ip := created.server.PublicNet.IPv4.IP; ip != nil {
		instance.IPv4 = ip.String()
	}

	if ip := created.server.PublicNet.IPv6.IP; ip != nil {
		instance.IPv6 = ip.String()
	}

	if instance.Domain == "" && instance.IPv4 != "" {
		instance.Domain = strings.ReplaceAll(instance.IPv4, ".", "-") + ".sslip.io"
	}

	readyErr := errors.New("server has no public IPv4 address")

	if instance.Domain != "" {
		readyErr = p.readiness.Wait(ctx, instance)
	}

	if readyErr != nil {
		// The caller's context may be what failed, so clean up on a fresh one.
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
		defer cancel()

		if err := p.DeleteInstance(cleanupCtx, instance.ServerID); err != nil {
			return nil, errors.Join(readyErr, fmt.Errorf("delete server %d: %w", instance.ServerID, err))
		}

		return nil, readyErr
	}

	return instance, nil
}

// DeleteInstance deletes the server backing an instance.
func (p *Provisioner) DeleteInstance(ctx context.Context, serverID int64) error {
	_, _, err := p.hcloud.Server.DeleteWithResult(ctx, &hcloud.Server{ID: serverID})

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
	stock, err := p.inStock(ctx)

	if err != nil {
		return nil, fmt.Errorf("list server availability: %w", err)
	}

	for _, serverType := range p.cfg.ServerTypes {
		for _, location := range p.cfg.Locations {
			if !stock[location][serverType] {
				continue
			}

			result, _, err := p.hcloud.Server.Create(ctx, p.serverCreateOpts(serverCreateOptsParams{
				name:       params.name,
				userData:   params.userData,
				serverType: serverType,
				location:   location,
			}))

			if hcloud.IsError(err, hcloud.ErrorCodeResourceUnavailable, hcloud.ErrorCodePlacementError) {
				continue
			}

			if err != nil {
				return nil, fmt.Errorf("create %s server in %s: %w", serverType, location, err)
			}

			if result.Server == nil {
				return nil, errors.New("hetzner returned no server")
			}

			return &createdServer{
				server:     result.Server,
				serverType: serverType,
				location:   location,
			}, nil
		}
	}

	return nil, provisioners.ErrNoCapacity
}

type serverCreateOptsParams struct {
	name       string
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
		Labels:           map[string]string{instanceLabel: params.name},
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

// inStock returns, per location name, the server type names that can be
// ordered there right now.
func (p *Provisioner) inStock(ctx context.Context) (map[string]map[string]bool, error) {
	serverTypes, err := p.hcloud.ServerType.All(ctx)

	if err != nil {
		return nil, err
	}

	names := map[int64]string{}

	for _, st := range serverTypes {
		names[st.ID] = st.Name
	}

	datacenters, err := p.hcloud.Datacenter.All(ctx)

	if err != nil {
		return nil, err
	}

	stock := map[string]map[string]bool{}

	for _, dc := range datacenters {
		if dc.Location == nil {
			continue
		}

		if stock[dc.Location.Name] == nil {
			stock[dc.Location.Name] = map[string]bool{}
		}

		for _, st := range dc.ServerTypes.Available {
			if name := names[st.ID]; name != "" {
				stock[dc.Location.Name][name] = true
			}
		}
	}

	return stock, nil
}
