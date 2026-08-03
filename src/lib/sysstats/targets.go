package sysstats

import (
	"slices"
	"sort"
	"strings"
)

// Target is a machine to scrape, along with the Stormkit services running on
// it. Several services commonly share one host, so they are grouped rather than
// scraped once each.
type Target struct {
	Host     string   `json:"host"`
	Services []string `json:"services"`

	// Manual marks a target that came from admin configuration rather than from
	// a Stormkit process registering itself.
	Manual bool `json:"manual"`
}

// RegisteredService is the subset of a service registry entry that target
// resolution needs. It keeps this package independent of rediscache, which
// matters because rediscache is what will consume it.
type RegisteredService struct {
	Name string
	Host string
}

type ResolveTargetsParams struct {
	Registered []RegisteredService
	Manual     []string
}

// ResolveTargets merges self-registered machines with manually configured ones,
// collapsing the services that share a host into a single target.
func ResolveTargets(p ResolveTargetsParams) []Target {
	byHost := map[string]*Target{}

	for _, svc := range p.Registered {
		host := normalizeHost(svc.Host)

		// A service that never advertised a host cannot be scraped. This is the
		// normal state for an instance running an older build.
		if host == "" {
			continue
		}

		target, ok := byHost[host]

		if !ok {
			target = &Target{Host: host}
			byHost[host] = target
		}

		if svc.Name != "" && !slices.Contains(target.Services, svc.Name) {
			target.Services = append(target.Services, svc.Name)
		}
	}

	for _, host := range p.Manual {
		host = normalizeHost(host)

		if host == "" {
			continue
		}

		// A manually listed host that also registered itself keeps its service
		// list; it is the same machine, described twice.
		if _, ok := byHost[host]; ok {
			continue
		}

		byHost[host] = &Target{Host: host, Manual: true}
	}

	out := make([]Target, 0, len(byHost))

	for _, target := range byHost {
		sort.Strings(target.Services)
		out = append(out, *target)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Host < out[j].Host
	})

	return out
}

func normalizeHost(host string) string {
	return strings.TrimSpace(host)
}
