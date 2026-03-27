package snippetshandlers

import (
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
)

// CalculateResetDomains returns the cache domains to reset when snippets change.
//
//   - nil        → reset all domains
//   - []string{} → reset nothing
//   - non-empty  → reset only those keys
func CalculateResetDomains(appDisplayName string, snippets []*buildconf.Snippet) []string {
	reset := map[string]bool{}

	for _, snippet := range snippets {
		if snippet.Rules == nil || len(snippet.Rules.Hosts) == 0 {
			return nil
		}

		for _, host := range snippet.Rules.Hosts {
			// It's enough that one rule contains no host configuration,
			// we'll have to reset cache for all domains anyways.
			if len(host) == 0 {
				return nil
			}

			if strings.EqualFold(host, "*.dev") {
				reset[appcache.DevDomainCacheKey(appDisplayName)] = true
			} else {
				reset[host] = true
			}

		}
	}

	if len(reset) == 0 {
		return []string{}
	}

	slice := make([]string, len(reset))
	count := 0

	for k := range reset {
		slice[count] = k
		count++
	}

	return slice
}

// NormalizeSnippetRules replaces individual stormkit dev hostnames with the
// wildcard "*.dev" token and lower-cases all other hosts.
func NormalizeSnippetRules(rules *buildconf.SnippetRule) {
	if rules == nil {
		return
	}

	hosts := []string{}
	added := false

	for _, host := range rules.Hosts {
		if appconf.IsStormkitDev(host) {
			if !added {
				hosts = append(hosts, "*.dev")
				added = true
			}
		} else {
			hosts = append(hosts, strings.ToLower(host))
		}
	}

	rules.Hosts = hosts
}

