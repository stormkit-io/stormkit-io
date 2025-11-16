//go:build alibaba

package integrations_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stretchr/testify/suite"
)

func setAlibabaEnvVars() {
	config.Get().Alibaba = &config.AlibabaConfig{
		Region: "me-central-1",
	}
}

type AlibabaSuite struct {
	suite.Suite
}

func (s *AlibabaSuite) SetupSuite() {
	setAlibabaEnvVars()
}

func (s *AlibabaSuite) BeforeTest(_, _ string) {
	integrations.CachedAlibabaClient = nil
}

func (s *AlibabaSuite) AfterTest(_, _ string) {
	integrations.CachedAlibabaClient = nil
}

func (s *AlibabaSuite) TearDownSuite() {
	config.Get().Alibaba = nil
}

func (s *AlibabaSuite) Test_Client() {
	client, err := integrations.Alibaba(integrations.ClientArgs{})
	s.NoError(err)
	s.NotNil(client)
}

func TestAlibabaClient(t *testing.T) {
	suite.Run(t, &AlibabaSuite{})
}
