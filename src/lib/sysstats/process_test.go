package sysstats

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ProcessSuite struct {
	suite.Suite
}

func (s *ProcessSuite) Test_CollectProcess() {
	stats := CollectProcess(CollectProcessParams{Service: "hosting", InstanceID: "abc"})

	s.Equal("hosting", stats.Service)
	s.Equal("abc", stats.InstanceID)
	s.Positive(stats.Goroutines)
	s.Positive(stats.HeapBytes)
}

// RSS is read from /proc, which only exists on Linux. Everywhere else the value
// is zero and callers fall back to the heap figure — it must not error or panic.
func (s *ProcessSuite) Test_CollectProcess_RSSIsOptional() {
	stats := CollectProcess(CollectProcessParams{})

	if runtime.GOOS == "linux" {
		s.Positive(stats.RSSBytes)
		return
	}

	s.Zero(stats.RSSBytes)
}

func TestProcessSuite(t *testing.T) {
	suite.Run(t, new(ProcessSuite))
}
