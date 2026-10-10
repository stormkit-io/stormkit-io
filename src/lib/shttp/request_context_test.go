package shttp_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type RemoteIPSuite struct {
	suite.Suite
}

func (s *RemoteIPSuite) AfterTest(_, _ string) {
	config.Get().TrustProxyHeaders = false
}

func (s *RemoteIPSuite) newReq(remoteAddr string, headers map[string]string) *shttp.RequestContext {
	h := http.Header{}

	for k, v := range headers {
		h.Set(k, v)
	}

	return &shttp.RequestContext{
		Request: &http.Request{
			RemoteAddr: remoteAddr,
			Header:     h,
		},
	}
}

func (s *RemoteIPSuite) Test_RemoteIP_SocketAddr() {
	req := s.newReq("1.2.3.4:5678", nil)

	s.Equal("1.2.3.4", req.RemoteIP())
	s.Equal("5678", req.RemotePort())
}

func (s *RemoteIPSuite) Test_RemoteIP_NoPort() {
	req := s.newReq("1.2.3.4", nil)

	s.Equal("1.2.3.4", req.RemoteIP())
	s.Equal("", req.RemotePort())
}

func (s *RemoteIPSuite) Test_RemoteIP_TrustProxy_XForwardedFor() {
	config.Get().TrustProxyHeaders = true

	req := s.newReq("10.0.0.1:9999", map[string]string{
		"X-Forwarded-For":  "1.2.3.4",
		"X-Forwarded-Port": "443",
	})

	s.Equal("1.2.3.4", req.RemoteIP())
	s.Equal("443", req.RemotePort())
}

func (s *RemoteIPSuite) Test_RemoteIP_TrustProxy_CommaSeparated() {
	config.Get().TrustProxyHeaders = true

	req := s.newReq("10.0.0.1:9999", map[string]string{
		"X-Forwarded-For": "1.2.3.4, 5.6.7.8, 9.10.11.12",
	})

	s.Equal("1.2.3.4", req.RemoteIP())
}

func (s *RemoteIPSuite) Test_RemoteIP_TrustProxy_NoProxyHeaders() {
	config.Get().TrustProxyHeaders = true

	req := s.newReq("1.2.3.4:5678", nil)

	s.Equal("1.2.3.4", req.RemoteIP())
	s.Equal("5678", req.RemotePort())
}

func (s *RemoteIPSuite) Test_RemoteIP_NoTrustProxy_IgnoresXFF() {
	req := s.newReq("1.2.3.4:5678", map[string]string{
		"X-Forwarded-For": "9.9.9.9",
	})

	s.Equal("1.2.3.4", req.RemoteIP())
	s.Equal("5678", req.RemotePort())
}

// Test_RemoteIP_NoTrustProxy_LoopbackUsesXFF verifies a loopback connection,
// which is this server proxying to a site it serves, carries the visitor's
// address even when proxy headers are not trusted.
func (s *RemoteIPSuite) Test_RemoteIP_NoTrustProxy_LoopbackUsesXFF() {
	for _, addr := range []string{"127.0.0.1:51234", "[::1]:51234"} {
		req := s.newReq(addr, map[string]string{
			"X-Forwarded-For":  "203.0.113.7",
			"X-Forwarded-Port": "40000",
		})

		s.Equal("203.0.113.7", req.RemoteIP(), addr)
		s.Equal("40000", req.RemotePort(), addr)
	}
}

func (s *RemoteIPSuite) Test_RemoteIP_NoTrustProxy_LoopbackWithoutXFF() {
	req := s.newReq("127.0.0.1:51234", nil)

	s.Equal("127.0.0.1", req.RemoteIP())
	s.Equal("51234", req.RemotePort())
}

func TestRemoteIPSuite(t *testing.T) {
	suite.Run(t, new(RemoteIPSuite))
}
