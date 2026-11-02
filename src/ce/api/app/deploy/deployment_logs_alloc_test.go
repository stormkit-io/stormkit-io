package deploy_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

// maxAllocPerLogByte bounds how many bytes rendering may allocate for each
// byte of stored log. Rendering copies the log a handful of times (split,
// step messages, JSON), so a linear implementation stays well under this.
const maxAllocPerLogByte = 20

type DeploymentLogsAllocSuite struct {
	suite.Suite
}

type legacyLogParams struct {
	lines     int
	lineBytes int
}

// legacyLog builds a text log in the format GitHub Actions runners store: a
// "[sk-step]" header followed by the step's build output, one line at a time.
func (s *DeploymentLogsAllocSuite) legacyLog(p legacyLogParams) string {
	var sb strings.Builder

	ts := time.Now().Unix()
	line := strings.Repeat("x", p.lineBytes-1) + "\n"

	sb.WriteString(fmt.Sprintf("[sk-step] npm ci [ts:%d]\n", ts))
	sb.WriteString("added 1200 packages in 12s\n")
	sb.WriteString(fmt.Sprintf("[sk-step] npm run build [ts:%d]\n", ts+12))

	for i := 0; i < p.lines; i++ {
		sb.WriteString(line)
	}

	sb.WriteString(fmt.Sprintf("[sk-step] [system] building finished [ts:%d]", ts+60))

	return sb.String()
}

// allocatedBy returns the bytes allocated on the heap while fn runs.
func (s *DeploymentLogsAllocSuite) allocatedBy(fn func()) uint64 {
	var before, after runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&before)
	fn()
	runtime.ReadMemStats(&after)

	return after.TotalAlloc - before.TotalAlloc
}

func (s *DeploymentLogsAllocSuite) Test_PrepareLogs_LegacyAllocationIsLinear() {
	for _, lines := range []int{5_000, 10_000, 20_000} {
		logs := s.legacyLog(legacyLogParams{lines: lines, lineBytes: 100})

		d := &deploy.Deployment{Logs: null.StringFrom(logs)}

		allocated := s.allocatedBy(func() {
			d.PrepareLogs(d.Logs.ValueOrZero(), false)
		})

		ratio := float64(allocated) / float64(len(logs))

		s.T().Logf("lines=%d log=%.1fMB allocated=%.1fMB ratio=%.0fx", lines, float64(len(logs))/1e6, float64(allocated)/1e6, ratio)

		s.LessOrEqualf(
			ratio,
			float64(maxAllocPerLogByte),
			"rendering a %d-line log allocated %.0fx its size", lines, ratio,
		)
	}
}

func TestDeploymentLogsAllocSuite(t *testing.T) {
	suite.Run(t, new(DeploymentLogsAllocSuite))
}
