package sysstats

import (
	"context"
	"os"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stretchr/testify/suite"
)

type ReporterSuite struct {
	suite.Suite
	ctx context.Context
}

func (s *ReporterSuite) SetupTest() {
	s.ctx = context.Background()
	os.Setenv("STORMKIT_SERVICE_NAME", "reporter-test")
	rediscache.DefaultService = nil
	rediscache.ResetService()
}

func (s *ReporterSuite) TearDownTest() {
	rediscache.Client().Del(s.ctx, rediscache.Service().Key(processStatsKey))
	os.Unsetenv("STORMKIT_SERVICE_NAME")
	rediscache.DefaultService = nil
	rediscache.ResetService()
}

func (s *ReporterSuite) Test_PublishAndRead() {
	publishProcessStats(s.ctx)

	byHost := ReadProcessStats()

	var found *ProcessStats

	for _, stats := range byHost {
		for i, stat := range stats {
			if stat.Service == "reporter-test" {
				found = &stats[i]
			}
		}
	}

	s.Require().NotNil(found, "this instance reports its own usage")
	s.Positive(found.Goroutines)
	s.Positive(found.HeapBytes)

	// Identity is attached from the registry entry rather than the payload, so
	// the two can never disagree.
	s.Equal("reporter-test", found.Service)
	s.NotEmpty(found.InstanceID)
}

func (s *ReporterSuite) Test_ReadWithNoReportsIsEmptyNotNil() {
	rediscache.Client().Del(s.ctx, rediscache.Service().Key(processStatsKey))

	s.NotNil(ReadProcessStats(), "callers index into this map directly")
}

func TestReporterSuite(t *testing.T) {
	suite.Run(t, new(ReporterSuite))
}
