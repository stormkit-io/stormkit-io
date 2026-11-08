package hosting

import (
	"os"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/libdns/hetzner/v2"
	"github.com/libdns/route53"
)

// hetznerChallengeTTL keeps challenge records short-lived, so a retry after a
// failed attempt isn't answered from a resolver's cache of the old one.
const hetznerChallengeTTL = 60 * time.Second

// ACMEDNSManager returns the DNS settings that solve DNS-01 challenges for
// the Stormkit Cloud wildcard certificate. Hetzner Cloud DNS is used when
// STORMKIT_ACME_HETZNER_TOKEN is set, Route53 otherwise, so the zone can move
// between them by changing the environment alone.
func ACMEDNSManager() certmagic.DNSManager {
	if token := os.Getenv("STORMKIT_ACME_HETZNER_TOKEN"); token != "" {
		return certmagic.DNSManager{
			DNSProvider: hetzner.New(token),
			TTL:         hetznerChallengeTTL,
		}
	}

	return certmagic.DNSManager{
		DNSProvider: &route53.Provider{
			AccessKeyId:     os.Getenv("STORMKIT_ACME_ACCESS_KEY"),
			SecretAccessKey: os.Getenv("STORMKIT_ACME_SECRET_KEY"),
			HostedZoneID:    os.Getenv("STORMKIT_DEV_ZONE_ID"),
		},
	}
}
