package hosting

import (
	"context"
	"fmt"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/libdns/libdns"
	"github.com/miekg/dns"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

// txtQuery asks one nameserver for the TXT values of a name.
type txtQuery struct {
	Nameserver string
	FQDN       string
}

// nameserverQuerier finds a zone's authoritative nameservers and asks them
// directly, bypassing any resolver cache.
type nameserverQuerier interface {
	nameservers(ctx context.Context, zone string) ([]string, error)
	txt(ctx context.Context, q txtQuery) ([]string, error)
}

// allNameserversProvider wraps a DNS provider so that adding a challenge
// record returns only once every authoritative nameserver serves it.
//
// certmagic checks propagation itself, but its check passes as soon as one
// nameserver has the record. Hetzner updates its nameservers one at a time, so
// Let's Encrypt could still ask one that lags behind and fail the challenge.
type allNameserversProvider struct {
	certmagic.DNSProvider

	querier  nameserverQuerier
	interval time.Duration
	timeout  time.Duration
}

// pendingTXT is a challenge record waiting to reach every nameserver.
type pendingTXT struct {
	Zone  string
	FQDN  string
	Value string
}

// AppendRecords adds the records, then waits for the TXT ones to be served by
// every nameserver of the zone. A record that is still missing somewhere when
// the wait times out is reported but not treated as an error: the challenge is
// worth trying anyway, certmagic retries it on failure, and it only cleans up
// records that were added without an error.
func (p *allNameserversProvider) AppendRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	added, err := p.DNSProvider.AppendRecords(ctx, zone, recs)

	if err != nil {
		return added, err
	}

	for _, rec := range recs {
		rr := rec.RR()

		if rr.Type != "TXT" {
			continue
		}

		pending := pendingTXT{Zone: zone, FQDN: libdns.AbsoluteName(rr.Name, zone), Value: rr.Data}

		if err := p.waitFor(ctx, pending); err != nil {
			slog.Errorf("dns challenge record %s is not on every nameserver: %v", pending.FQDN, err)
		}
	}

	return added, nil
}

func (p *allNameserversProvider) waitFor(ctx context.Context, rec pendingTXT) error {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	missing, err := p.querier.nameservers(ctx, rec.Zone)

	if err != nil {
		return fmt.Errorf("looking up nameservers of %s: %w", rec.Zone, err)
	}

	if len(missing) == 0 {
		return fmt.Errorf("no nameservers found for %s", rec.Zone)
	}

	for {
		missing = p.missingOn(ctx, rec, missing)

		if len(missing) == 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("%w; still missing on %s", ctx.Err(), strings.Join(missing, ", "))
		case <-time.After(p.interval):
		}
	}
}

// missingOn returns the nameservers that don't serve the record yet. A
// nameserver that fails to answer counts as missing and is asked again.
func (p *allNameserversProvider) missingOn(ctx context.Context, rec pendingTXT, nameservers []string) []string {
	var missing []string

	for _, ns := range nameservers {
		values, err := p.querier.txt(ctx, txtQuery{Nameserver: ns, FQDN: rec.FQDN})

		if err != nil || !slices.Contains(values, rec.Value) {
			missing = append(missing, ns)
		}
	}

	return missing
}

// authoritativeDNS queries nameservers over the network.
type authoritativeDNS struct {
	client *dns.Client
}

func (a authoritativeDNS) nameservers(ctx context.Context, zone string) ([]string, error) {
	records, err := net.DefaultResolver.LookupNS(ctx, zone)

	if err != nil {
		return nil, err
	}

	hosts := make([]string, 0, len(records))

	for _, record := range records {
		hosts = append(hosts, net.JoinHostPort(strings.TrimSuffix(record.Host, "."), "53"))
	}

	return hosts, nil
}

func (a authoritativeDNS) txt(ctx context.Context, q txtQuery) ([]string, error) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(q.FQDN), dns.TypeTXT)
	msg.RecursionDesired = false

	res, _, err := a.client.ExchangeContext(ctx, msg, q.Nameserver)

	if err != nil {
		return nil, err
	}

	var values []string

	for _, rr := range res.Answer {
		if txt, ok := rr.(*dns.TXT); ok {
			values = append(values, strings.Join(txt.Txt, ""))
		}
	}

	return values, nil
}
