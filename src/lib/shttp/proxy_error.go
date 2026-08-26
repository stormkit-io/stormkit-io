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
	Target  string
	After   time.Duration
	Wrapped error
}

func (e *ProxyTimeoutError) Error() string {
	return fmt.Sprintf(
		"upstream did not send response headers within %s (%s). The server may still be processing the request — raise %s if this endpoint is expected to take longer.",
		e.After, ProxyTimeoutEnvVar, ProxyTimeoutEnvVar,
	)
}

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
