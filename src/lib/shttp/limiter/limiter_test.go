package limiter_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/limiter"
	"github.com/stretchr/testify/suite"
)

type IPSuite struct {
	suite.Suite
}

func (s *IPSuite) TearDownTest() {
	config.Get().TrustProxyHeaders = false
}

func (s *IPSuite) Test_EdgeMode_UsesRemoteAddr() {
	hdr := http.Header{}
	hdr.Set("X-Forwarded-For", "9.9.9.9")
	hdr.Set("X-Real-IP", "8.8.8.8")

	req := &http.Request{
		RemoteAddr: "1.2.3.4:1234",
		Header:     hdr,
	}

	s.Equal("1.2.3.4", limiter.IP(req))
}

func (s *IPSuite) Test_EdgeMode_FallsBackToRemoteAddrWithoutPort() {
	req := &http.Request{
		RemoteAddr: "127.0.0.1",
	}

	s.Equal("127.0.0.1", limiter.IP(req))
}

func (s *IPSuite) Test_TrustProxy_UsesXForwardedFor() {
	config.Get().TrustProxyHeaders = true

	hdr := http.Header{}
	hdr.Set("X-Forwarded-For", "1.1.1.1")

	req := &http.Request{
		RemoteAddr: "10.0.0.1:1234",
		Header:     hdr,
	}

	s.Equal("1.1.1.1", limiter.IP(req))
}

func (s *IPSuite) Test_TrustProxy_FallsBackToXRealIP() {
	config.Get().TrustProxyHeaders = true

	hdr := http.Header{}
	hdr.Set("X-Real-IP", "127.0.0.1")

	req := &http.Request{
		RemoteAddr: "10.0.0.1:1234",
		Header:     hdr,
	}

	s.Equal("127.0.0.1", limiter.IP(req))
}

func (s *IPSuite) Test_TrustProxy_FallsBackToRemoteAddr() {
	config.Get().TrustProxyHeaders = true

	req := &http.Request{
		RemoteAddr: "10.0.0.1:5000",
	}

	s.Equal("10.0.0.1", limiter.IP(req))
}

func TestIPSuite(t *testing.T) {
	suite.Run(t, new(IPSuite))
}
