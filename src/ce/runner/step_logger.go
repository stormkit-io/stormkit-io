package runner

import (
	"strings"
	"sync"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
)

// stepLogger accumulates the build output as structured steps. Command
// output streams into the step that is current at write time, mirroring how
// the legacy text format interleaved output with step markers. The logger
// itself is a stable io.Writer, so call sites can hold on to it across step
// boundaries.
type stepLogger struct {
	mu    sync.Mutex
	steps []*loggedStep
}

type loggedStep struct {
	title      string
	startedAt  int64
	finishedAt int64
	status     string
	buf        *CustomBuffer
}

func newStepLogger() *stepLogger {
	return &stepLogger{}
}

// Write streams command output into the current step. Output arriving
// before the first step is dropped, matching the legacy parser which
// ignored lines preceding the first step marker.
func (l *stepLogger) Write(p []byte) (int, error) {
	l.mu.Lock()

	var buf *CustomBuffer

	if len(l.steps) > 0 {
		buf = l.steps[len(l.steps)-1].buf
	}

	l.mu.Unlock()

	if buf == nil {
		return len(p), nil
	}

	return buf.Write(p)
}

// addStep closes the current step as successful and opens a new one. Titles
// prefixed with "[system] " are markers rather than steps: they close the
// current step without opening a new one. Empty titles are treated the same
// way — ParseStepLogs skips untitled records.
func (l *stepLogger) addStep(title string) {
	now := time.Now().UnixMilli()

	l.mu.Lock()
	defer l.mu.Unlock()

	l.closeCurrent(now, deploy.StepStatusSuccess)

	if title == "" || strings.HasPrefix(title, "[system] ") {
		return
	}

	l.steps = append(l.steps, &loggedStep{
		title:     title,
		startedAt: now,
		buf:       NewCustomBuffer(),
	})
}

// closeStep closes the current step with an explicit outcome.
func (l *stepLogger) closeStep(success bool) {
	status := deploy.StepStatusSuccess

	if !success {
		status = deploy.StepStatusFailed
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.closeCurrent(time.Now().UnixMilli(), status)
}

// closeCurrent stamps the current step's finish time and status. A step
// that is already closed is left untouched.
func (l *stepLogger) closeCurrent(finishedAt int64, status string) {
	if len(l.steps) == 0 {
		return
	}

	current := l.steps[len(l.steps)-1]

	if current.finishedAt != 0 {
		return
	}

	current.finishedAt = finishedAt
	current.status = status
}

// marshal serializes the steps into the structured NDJSON log format.
func (l *stepLogger) marshal() string {
	l.mu.Lock()
	defer l.mu.Unlock()

	records := make([]deploy.StepRecord, 0, len(l.steps))

	for _, step := range l.steps {
		records = append(records, deploy.StepRecord{
			Title:      step.title,
			Message:    step.buf.String(),
			StartedAt:  step.startedAt,
			FinishedAt: step.finishedAt,
			Status:     step.status,
		})
	}

	return deploy.MarshalStepLogs(records)
}
