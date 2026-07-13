package limiter

import (
	"net"
	"net/http"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
)

// Options represents the rate limit options.
type Options struct {
	// Limit is the number of requests that are limited
	// within a given time duration.
	Limit int64

	// Burst is the number of tokens that will be generated
	// every 1 / limit second. User will accumulate these tokens
	// when there is no active connection. On every request,
	// these tokens will be consumed. Once consumed, the
	// user will receive a 429 request.
	Burst int

	// Duration is the time in which the rate limiter
	// should operate.
	Duration time.Duration

	// Hash specifies the parts of the request to include
	// in the rate-limiting key. Possible values are:
	// ip, path, header:<my-header-1,my-header-2>,
	Hash []string
}

func IP(r *http.Request) string {
	if r == nil {
		return ""
	}

	if config.Get().TrustProxyHeaders {
		if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			return forwardedFor
		}

		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			return realIP
		}
	}

	return getRemoteAddr(r)
}

// getRemoteIpAddr, extract port
func getRemoteAddr(r *http.Request) string {
	// Get the IP address from r.RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// Fallback to the RemoteAddr if parsing fails (e.g., no port in the address)
		return r.RemoteAddr
	}

	// Check if the IP address is an IPv6 address
	ip := net.ParseIP(host)
	if ip == nil {
		// Not a valid IP address, return the original host
		return host
	}

	// Check if the IP address is IPv4-mapped IPv6 address (::ffff:x.x.x.x)
	if ipv4 := ip.To4(); ipv4 != nil {
		// It's an IPv4-mapped IPv6 address, return the IPv4 part only
		return ipv4.String()
	}

	// It's a pure IPv6 address, return it
	return ip.String()
}
