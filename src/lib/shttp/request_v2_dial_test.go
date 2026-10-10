package shttp

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stretchr/testify/suite"
)

type RequestV2DialSuite struct {
	suite.Suite

	server       *httptest.Server
	hits         atomic.Int32
	host         atomic.Value
	serverName   atomic.Value
	headers      atomic.Value
	originalTLS  *tls.Config
	redirectedTo string
}

func (s *RequestV2DialSuite) SetupTest() {
	s.hits.Store(0)
	s.redirectedTo = ""

	s.server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.hits.Add(1)
		s.host.Store(r.Host)
		s.serverName.Store(r.TLS.ServerName)
		s.headers.Store(r.Header.Clone())

		if s.redirectedTo != "" {
			http.Redirect(w, r, s.redirectedTo, http.StatusFound)
			return
		}

		_, _ = io.WriteString(w, "served "+r.URL.Path)
	}))

	// Trust the test server's certificate on the dial transport only.
	pool := x509.NewCertPool()
	pool.AddCert(s.server.Certificate())

	s.originalTLS = transports.dial.TLSClientConfig
	transports.dial.TLSClientConfig = &tls.Config{RootCAs: pool}
}

func (s *RequestV2DialSuite) TearDownTest() {
	s.server.Close()
	transports.dial.TLSClientConfig = s.originalTLS
	transports.dial.CloseIdleConnections()
}

// Test_DialAddr_KeepsHostAndServerName verifies the connection goes to the
// override address while the Host header and TLS server name stay the target's.
func (s *RequestV2DialSuite) Test_DialAddr_KeepsHostAndServerName() {
	res, err := NewRequestV2(http.MethodGet, "https://example.com/landing/index.html").
		DialAddr(s.server.Listener.Addr().String()).
		Do()

	s.Require().NoError(err)

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	s.Require().NoError(err)

	s.Equal(http.StatusOK, res.StatusCode)
	s.Equal("served /landing/index.html", string(body))
	s.Equal("example.com", s.host.Load())
	s.Equal("example.com", s.serverName.Load())
}

// Test_DialAddr_RedirectToOtherHostDialsNormally verifies a redirect to a
// different host is not sent to the override address.
func (s *RequestV2DialSuite) Test_DialAddr_RedirectToOtherHostDialsNormally() {
	other := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "other "+r.Host)
	}))

	defer other.Close()

	s.redirectedTo = other.URL + "/elsewhere"

	res, err := NewRequestV2(http.MethodGet, "https://example.com/start").
		DialAddr(s.server.Listener.Addr().String()).
		Do()

	s.Require().NoError(err)

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	s.Require().NoError(err)

	s.Equal(int32(1), s.hits.Load())
	s.True(strings.HasPrefix(string(body), "other 127.0.0.1:"))
}

// proxyRequest returns a request from visitor 198.51.100.9:40000 with the
// given headers.
func (s *RequestV2DialSuite) proxyRequest(headers map[string]string) *RequestContext {
	r := httptest.NewRequest(http.MethodGet, "https://site.stormkit.dev/landing/", nil)
	r.RemoteAddr = "198.51.100.9:40000"

	for k, v := range headers {
		r.Header.Set(k, v)
	}

	return &RequestContext{Request: r}
}

// Test_Proxy_LocalDial_ForwardsVisitor verifies a loopback hop carries the
// visitor's address.
func (s *RequestV2DialSuite) Test_Proxy_LocalDial_ForwardsVisitor() {
	res := Proxy(s.proxyRequest(nil), ProxyArgs{
		Target:   "https://example.com/landing/",
		DialAddr: s.server.Listener.Addr().String(),
	})

	s.Equal(http.StatusOK, res.Status)

	h := s.headers.Load().(http.Header)

	s.Equal("198.51.100.9", h.Get("X-Forwarded-For"))
	s.Equal("40000", h.Get("X-Forwarded-Port"))
}

// Test_Proxy_LocalDial_TrustedChainUnchanged verifies a chain from a trusted
// upstream proxy reaches the target as it would on a direct request.
func (s *RequestV2DialSuite) Test_Proxy_LocalDial_TrustedChainUnchanged() {
	config.Get().TrustProxyHeaders = true

	defer func() { config.Get().TrustProxyHeaders = false }()

	res := Proxy(s.proxyRequest(map[string]string{"X-Forwarded-For": "203.0.113.7, 10.0.0.5"}), ProxyArgs{
		Target:   "https://example.com/landing/",
		DialAddr: s.server.Listener.Addr().String(),
	})

	s.Equal(http.StatusOK, res.Status)
	s.Equal("203.0.113.7, 10.0.0.5", s.headers.Load().(http.Header).Get("X-Forwarded-For"))
}

func TestRequestV2DialSuite(t *testing.T) {
	suite.Run(t, new(RequestV2DialSuite))
}
