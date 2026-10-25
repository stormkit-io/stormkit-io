package utils

import (
	"fmt"
	"net"
	"net/netip"
	"strings"
)

// internalHopPrefixes are ranges that never belong to a visitor on the public
// internet: private, loopback, link-local, CGNAT (used by some cloud load
// balancers, Tailscale and Kubernetes networks) and other non-routable space.
var internalHopPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("::/128"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("ff00::/8"),
}

// ForwardedClientIPParams configures ForwardedClientIP.
type ForwardedClientIPParams struct {
	// Header is the X-Forwarded-For style comma-separated list.
	Header string

	// TrustedProxies are ranges of proxies in front of Stormkit whose entries
	// are skipped like internal hops (e.g. a CDN's published ranges).
	TrustedProxies []netip.Prefix
}

// ForwardedClientIP returns the client address from an X-Forwarded-For style
// list. Each proxy appends the address it received the request from, so only
// the right-most entries were written by trusted infrastructure and anything
// to their left may be client-supplied. Walking right to left, it skips
// internal hops (private, loopback, link-local, CGNAT and other non-routable
// addresses) and trusted proxies, and returns the first remaining address.
// When every entry is skipped, as on an intranet-only install, it returns the
// left-most address, which is the original client. A port on an entry is
// dropped.
func ForwardedClientIP(p ForwardedClientIPParams) string {
	chain := forwardedChain{trusted: p.TrustedProxies}
	entries := chain.entries(p.Header)

	for i := len(entries) - 1; i >= 0; i-- {
		if addr, err := netip.ParseAddr(entries[i]); err == nil && !chain.isSkipped(addr) {
			return entries[i]
		}
	}

	for _, entry := range entries {
		if _, err := netip.ParseAddr(entry); err == nil {
			return entry
		}
	}

	if len(entries) > 0 {
		return entries[0]
	}

	return ""
}

// forwardedChain classifies the entries of an X-Forwarded-For list.
type forwardedChain struct {
	trusted []netip.Prefix
}

// entries returns the non-empty entries of the list, trimmed and without ports.
func (c forwardedChain) entries(header string) []string {
	parts := strings.Split(header, ",")
	entries := make([]string, 0, len(parts))

	for _, part := range parts {
		entry := strings.TrimSpace(part)

		if host, _, err := net.SplitHostPort(entry); err == nil {
			entry = host
		}

		if entry != "" {
			entries = append(entries, entry)
		}
	}

	return entries
}

// isSkipped reports whether addr is an internal hop or a trusted proxy rather
// than the client.
func (c forwardedChain) isSkipped(addr netip.Addr) bool {
	addr = addr.Unmap().WithZone("")

	for _, prefix := range internalHopPrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}

	for _, prefix := range c.trusted {
		if prefix.Contains(addr) {
			return true
		}
	}

	return false
}

// IsPortInUse checks if the given port is currently in use by any process.
func IsPortInUse(port int) bool {
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))

	if err != nil {
		// Port is not in use
		return false
	}

	if conn != nil {
		conn.Close()
	}

	return true
}
