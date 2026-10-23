package analytics_test

import (
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type CampaignSuite struct {
	suite.Suite
}

func (s *CampaignSuite) Test_ReadsUTMAndRef() {
	c := analytics.CampaignFromRawQuery("?utm_source=x&utm_medium=social&utm_campaign=Launch%20Day&ref=indiefounders&page=2")

	s.Equal(analytics.Campaign{
		Source: null.StringFrom("x"),
		Medium: null.StringFrom("social"),
		Name:   null.StringFrom("Launch Day"),
		Ref:    null.StringFrom("indiefounders"),
	}, c)
}

func (s *CampaignSuite) Test_AbsentOrBlankValuesAreInvalid() {
	c := analytics.CampaignFromRawQuery("utm_source=%20%20&utm_medium=&page=2")

	s.Equal(analytics.Campaign{}, c)
	s.Equal(analytics.Campaign{}, analytics.CampaignFromRawQuery(""))
	s.Equal(analytics.Campaign{}, analytics.CampaignFromRawQuery("%zz"))
}

func (s *CampaignSuite) Test_SanitizesValues() {
	long := strings.Repeat("ü", 300)
	c := analytics.CampaignFromRawQuery("utm_source=a%00b&utm_campaign=" + long + "&ref=%ff")

	s.Equal("ab", c.Source.ValueOrZero())
	s.Equal(strings.Repeat("ü", 200), c.Name.ValueOrZero())
	s.False(c.Ref.Valid, "invalid UTF-8 is dropped")
}

func TestCampaignSuite(t *testing.T) {
	suite.Run(t, new(CampaignSuite))
}
