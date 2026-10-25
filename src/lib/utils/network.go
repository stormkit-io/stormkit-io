package utils

import (
	"fmt"
	"net"
	"strings"
)

// ForwardedClientIP returns the client address from an X-Forwarded-For style
// list. Each proxy appends the address it received the request from, so only
// the right-most entries were written by trusted infrastructure and anything
// to their left may be client-supplied. It returns the right-most public
// address, skipping private and loopback hops added by internal proxies, or
// the right-most entry when none is public. A port on an entry is dropped.
func ForwardedClientIP(header string) string {
	entries := strings.Split(header, ",")
	rightMost := ""

	for i := len(entries) - 1; i >= 0; i-- {
		entry := strings.TrimSpace(entries[i])

		if host, _, err := net.SplitHostPort(entry); err == nil {
			entry = host
		}

		if entry == "" {
			continue
		}

		if rightMost == "" {
			rightMost = entry
		}

		ip := net.ParseIP(entry)

		if ip != nil && ip.IsGlobalUnicast() && !ip.IsPrivate() {
			return entry
		}
	}

	return rightMost
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
