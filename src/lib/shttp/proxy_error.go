package shttp

import (
	"errors"
	"fmt"
	"net"
	"time"
)

// ProxyTimeoutEnvVar is the environment variable that controls how long the
// proxy waits for an upstream to start sending response headers.
const ProxyTimeoutEnvVar = "STORMKIT_HTTP_PROXY_TIMEOUT"

// ProxyTimeoutError reports that an upstream did not start sending response
// headers within the configured proxy timeout.
//
// It exists so the failure is distinguishable from an application crash: the
// upstream process is usually alive and still working when the proxy gives up,
// and the fix is a configuration change rather than a code change. Callers
// match on it with errors.As to answer 504 and name the knob.
type ProxyTimeoutError struct {
	// Target is the upstream that was being proxied. It is an internal
	// host:port, so it belongs in logs and never in a rendered page.
	Target string
	// After is how long the proxy actually waited before giving up.
	After time.Duration
	// Limit is the configured deadline that fired.
	Limit   time.Duration
	Wrapped error
}

// Error states what happened and nothing else. The remedy belongs to whoever
// renders this — the hosting error page already spells it out, and repeating it
// here printed the same advice twice on the same page.
//
// Target is deliberately left out: this string is rendered into a page served
// to a deployment's visitors, and the target is an internal host:port.
func (e *ProxyTimeoutError) Error() string {
	return fmt.Sprintf("upstream did not send response headers within %s (%s)", e.Limit, ProxyTimeoutEnvVar)
}

// Unwrap exposes the transport error, so a caller can still inspect the
// underlying *url.Error or net.Error.
func (e *ProxyTimeoutError) Unwrap() error {
	return e.Wrapped
}

// Timeout mirrors net.Error's method so a caller holding this error can ask the
// question directly rather than unwrapping to the transport's error first.
func (e *ProxyTimeoutError) Timeout() bool {
	return true
}

// isTimeout reports whether err is a transport deadline — the overall client
// timeout on a buffered request or ResponseHeaderTimeout on a streaming one.
// Both surface as a *url.Error whose Timeout() is true.
func isTimeout(err error) bool {
	var netErr net.Error

	return errors.As(err, &netErr) && netErr.Timeout()
}
