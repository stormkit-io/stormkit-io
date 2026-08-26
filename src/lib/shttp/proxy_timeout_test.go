package shttp_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

// ProxyTimeoutSuite covers the distinct upstream-timeout failure (issue #421).
// A slow upstream used to surface as a 500 carrying a raw transport error,
// indistinguishable from an application crash.
type ProxyTimeoutSuite struct {
	suite.Suite

	previousTimeout time.Duration
}

func (s *ProxyTimeoutSuite) SetupTest() {
	s.previousTimeout = config.Get().HTTPTimeouts.ProxyTimeout
	config.Get().HTTPTimeouts.ProxyTimeout = 100 * time.Millisecond
}

func (s *ProxyTimeoutSuite) TearDownTest() {
	config.Get().HTTPTimeouts.ProxyTimeout = s.previousTimeout
}

func (s *ProxyTimeoutSuite) proxy(target string) *shttp.Response {
	req, err := http.NewRequest(http.MethodGet, target, nil)
	s.Require().NoError(err)

	return shttp.Proxy(shttp.NewRequestContext(req), shttp.ProxyArgs{Target: target})
}

func (s *ProxyTimeoutSuite) Test_SlowUpstreamYieldsGatewayTimeout() {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	defer upstream.Close()

	res := s.proxy(upstream.URL)

	s.Equal(http.StatusGatewayTimeout, res.Status)

	var timeoutErr *shttp.ProxyTimeoutError

	s.Require().True(errors.As(res.Error, &timeoutErr), "expected a ProxyTimeoutError, got %v", res.Error)
	s.Equal(100*time.Millisecond, timeoutErr.Limit, "the reported deadline must be the one that was enforced")
	s.Equal(upstream.URL, timeoutErr.Target)
	s.True(timeoutErr.Timeout())

	// The message has to name the knob, otherwise tracing the failure back to
	// the proxy is a manual investigation.
	s.Contains(timeoutErr.Error(), shttp.ProxyTimeoutEnvVar)
	s.Contains(timeoutErr.Error(), "did not send response headers")

	body, ok := res.Data.(string)
	s.Require().True(ok)
	s.Contains(body, shttp.ProxyTimeoutEnvVar)
}

func (s *ProxyTimeoutSuite) Test_UnwrapsToTheTransportError() {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(500 * time.Millisecond)
	}))

	defer upstream.Close()

	res := s.proxy(upstream.URL)

	var urlErr *url.Error

	s.Require().True(errors.As(res.Error, &urlErr), "the transport error must stay reachable for callers that inspect it")
	s.True(urlErr.Timeout())
}

func (s *ProxyTimeoutSuite) Test_NonTimeoutFailureStaysAServerError() {
	// A closed port fails to connect rather than timing out, so it must not be
	// reported as a timeout and must not name the timeout env var.
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	target := upstream.URL
	upstream.Close()

	res := s.proxy(target)

	s.Equal(http.StatusInternalServerError, res.Status)

	var timeoutErr *shttp.ProxyTimeoutError

	s.False(errors.As(res.Error, &timeoutErr))

	body, _ := res.Data.(string)
	s.False(strings.Contains(body, shttp.ProxyTimeoutEnvVar))
}

func (s *ProxyTimeoutSuite) Test_HealthyUpstreamIsUnaffected() {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	defer upstream.Close()

	res := s.proxy(upstream.URL)

	s.Equal(http.StatusOK, res.Status)
	s.Nil(res.Error)
	s.Equal([]byte("ok"), res.Data)
}

func (s *ProxyTimeoutSuite) Test_TargetIsNotRenderedIntoTheBody() {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	defer upstream.Close()

	res := s.proxy(upstream.URL)

	// The body reaches a deployment's visitors; the upstream is an internal
	// host:port and must stay in the logs.
	body, _ := res.Data.(string)
	s.NotContains(body, upstream.URL)
}

func (s *ProxyTimeoutSuite) Test_DisabledDeadlineStillFailsAsAServerError() {
	// `0` disables the deadline, and the runtime docs offer it for long
	// requests. Nothing under that setting may be reported as a proxy timeout.
	//
	// This covers the connection failure only. The case the `timeout > 0` guard
	// in Proxy exists for — a dial or TLS-handshake timeout, which isTimeout
	// also matches but STORMKIT_HTTP_PROXY_TIMEOUT does not govern — needs a
	// blackholed route and the transport's own 30s dialer deadline, so it is not
	// reproducible cheaply here.
	config.Get().HTTPTimeouts.ProxyTimeout = 0

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	target := upstream.URL
	upstream.Close()

	res := s.proxy(target)

	var timeoutErr *shttp.ProxyTimeoutError

	s.False(errors.As(res.Error, &timeoutErr))
	s.Equal(http.StatusInternalServerError, res.Status)

	body, _ := res.Data.(string)
	s.False(strings.Contains(body, "0s"), "must never tell the operator the deadline was 0s")
}

func TestProxyTimeoutSuite(t *testing.T) {
	suite.Run(t, new(ProxyTimeoutSuite))
}
