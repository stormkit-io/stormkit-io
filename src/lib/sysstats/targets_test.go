package sysstats

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type TargetsSuite struct {
	suite.Suite
}

func (s *TargetsSuite) Test_GroupsServicesSharingAHost() {
	targets := ResolveTargets(ResolveTargetsParams{
		Registered: []RegisteredService{
			{Name: "hosting", Host: "node-a"},
			{Name: "workerserver", Host: "node-a"},
			{Name: "hosting", Host: "node-b"},
		},
	})

	s.Require().Len(targets, 2, "one target per machine, not per service")
	s.Equal("node-a", targets[0].Host)
	s.Equal([]string{"hosting", "workerserver"}, targets[0].Services)
	s.Equal("node-b", targets[1].Host)
	s.Equal([]string{"hosting"}, targets[1].Services)
}

func (s *TargetsSuite) Test_DeduplicatesRepeatedServiceNames() {
	targets := ResolveTargets(ResolveTargetsParams{
		Registered: []RegisteredService{
			{Name: "hosting", Host: "node-a"},
			{Name: "hosting", Host: "node-a"},
		},
	})

	s.Require().Len(targets, 1)
	s.Equal([]string{"hosting"}, targets[0].Services)
}

// An instance running an older build advertises no host. It cannot be scraped,
// but it must not produce a bogus target either.
func (s *TargetsSuite) Test_SkipsServicesWithoutAHost() {
	targets := ResolveTargets(ResolveTargetsParams{
		Registered: []RegisteredService{
			{Name: "hosting", Host: ""},
			{Name: "workerserver", Host: "  "},
			{Name: "hosting", Host: "node-a"},
		},
	})

	s.Require().Len(targets, 1)
	s.Equal("node-a", targets[0].Host)
}

func (s *TargetsSuite) Test_IncludesManualTargets() {
	targets := ResolveTargets(ResolveTargetsParams{
		Registered: []RegisteredService{{Name: "hosting", Host: "node-a"}},
		Manual:     []string{"db-host:9100"},
	})

	s.Require().Len(targets, 2)

	s.Equal("db-host:9100", targets[0].Host)
	s.True(targets[0].Manual)
	s.Empty(targets[0].Services, "a machine with no Stormkit process runs no services")

	s.Equal("node-a", targets[1].Host)
	s.False(targets[1].Manual)
}

// The same machine listed manually and registering itself is one machine.
func (s *TargetsSuite) Test_ManualTargetDoesNotDuplicateRegisteredHost() {
	targets := ResolveTargets(ResolveTargetsParams{
		Registered: []RegisteredService{{Name: "hosting", Host: "node-a"}},
		Manual:     []string{"node-a"},
	})

	s.Require().Len(targets, 1)
	s.Equal([]string{"hosting"}, targets[0].Services)
	s.False(targets[0].Manual, "the registered entry is the richer one")
}

func (s *TargetsSuite) Test_EmptyInput() {
	s.Empty(ResolveTargets(ResolveTargetsParams{}))
}

func TestTargetsSuite(t *testing.T) {
	suite.Run(t, new(TargetsSuite))
}
