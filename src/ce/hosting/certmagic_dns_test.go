package hosting_test

import (
	"testing"
	"time"

	"github.com/libdns/hetzner/v2"
	"github.com/libdns/route53"
	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stretchr/testify/suite"
)

type CertmagicDNSSuite struct {
	suite.Suite
}

func (s *CertmagicDNSSuite) Test_Route53_ByDefault() {
	s.T().Setenv("STORMKIT_ACME_HETZNER_TOKEN", "")
	s.T().Setenv("STORMKIT_ACME_ACCESS_KEY", "access-key")
	s.T().Setenv("STORMKIT_ACME_SECRET_KEY", "secret-key")
	s.T().Setenv("STORMKIT_DEV_ZONE_ID", "Z123")

	manager := hosting.ACMEDNSManager()
	inner, wrapped := hosting.UnwrapDNSProvider(manager.DNSProvider)
	provider, ok := inner.(*route53.Provider)

	s.False(wrapped)
	s.Require().True(ok)
	s.Equal("access-key", provider.AccessKeyId)
	s.Equal("secret-key", provider.SecretAccessKey)
	s.Equal("Z123", provider.HostedZoneID)
	s.Zero(manager.TTL)
}

// Test_Hetzner_WhenTokenIsSet verifies that a Hetzner token takes precedence
// over leftover Route53 credentials, so setting it alone moves the challenges,
// and that Hetzner records are waited on until every nameserver serves them.
func (s *CertmagicDNSSuite) Test_Hetzner_WhenTokenIsSet() {
	s.T().Setenv("STORMKIT_ACME_HETZNER_TOKEN", "hetzner-token")
	s.T().Setenv("STORMKIT_ACME_ACCESS_KEY", "access-key")

	manager := hosting.ACMEDNSManager()
	inner, wrapped := hosting.UnwrapDNSProvider(manager.DNSProvider)
	provider, ok := inner.(*hetzner.Provider)

	s.True(wrapped)
	s.Require().True(ok)
	s.Equal("hetzner-token", provider.APIToken)
	s.Equal(60*time.Second, manager.TTL)
}

func TestCertmagicDNSSuite(t *testing.T) {
	suite.Run(t, new(CertmagicDNSSuite))
}
