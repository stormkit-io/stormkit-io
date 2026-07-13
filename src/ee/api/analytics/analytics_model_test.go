package analytics_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stretchr/testify/suite"
)

type AnalyticsModelSuite struct {
	suite.Suite
}

func (s *AnalyticsModelSuite) Test_ExtractHostnameFromReferrer() {
	tests := []struct {
		referrer string
		expected string
	}{
		{
			referrer: "https://example.com/path/to/page",
			expected: "example.com",
		},
		{
			referrer: "http://subdomain.example.org/another/path",
			expected: "subdomain.example.org",
		},
		{
			referrer: "https://www.test-site.co.uk/",
			expected: "www.test-site.co.uk",
		},
		{
			referrer: "invalid-url",
			expected: "invalid-url", // We don't want to return empty for invalid URLs
		},
		{
			referrer: "example.com",
			expected: "example.com",
		},
		{
			referrer: "",
			expected: "",
		},
	}

	for _, test := range tests {
		result := analytics.NormalizeReferrer(test.referrer)
		s.Equal(test.expected, result)
	}
}

func TestAnalyticsModel(t *testing.T) {
	suite.Run(t, new(AnalyticsModelSuite))
}
