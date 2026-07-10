package deploy

import (
	"encoding/json"
	"strings"
)

const StepStatusSuccess = "success"
const StepStatusFailed = "failed"

// StepRecord is one deployment step in the structured log format. Logs in
// this format are stored as NDJSON - one record per line - so a step can be
// appended to the stored blob without re-parsing it. Timestamps are unix
// milliseconds; a zero FinishedAt marks a step that is still running.
//
// The legacy format (raw build output interleaved with "[sk-step]" markers)
// is still written by GitHub-action builds and read from old deployments.
// PrepareLogs detects the format of the stored blob and picks the parser.
type StepRecord struct {
	Title      string         `json:"title"`
	Message    string         `json:"message,omitempty"`
	StartedAt  int64          `json:"startedAt"`
	FinishedAt int64          `json:"finishedAt,omitempty"`
	Status     string         `json:"status,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}

// ParseStepLogs decodes an NDJSON log blob into step records. It returns
// false when the blob is not in the structured format. A trailing line that
// fails to decode is skipped rather than failing the whole blob, so a
// partially written last record does not hide the completed steps.
func ParseStepLogs(rawLogs string) ([]StepRecord, bool) {
	trimmed := strings.TrimSpace(rawLogs)

	if !strings.HasPrefix(trimmed, "{") {
		return nil, false
	}

	steps := []StepRecord{}

	for _, line := range strings.Split(trimmed, "\n") {
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		var step StepRecord

		if err := json.Unmarshal([]byte(line), &step); err != nil {
			continue
		}

		// A JSON line without a title is not a step record (e.g. JSON build
		// output); skip it rather than rejecting the whole blob.
		if step.Title == "" {
			continue
		}

		steps = append(steps, step)
	}

	if len(steps) == 0 {
		return nil, false
	}

	return steps, true
}

// MarshalStepLogs encodes step records as an NDJSON log blob. HTML escaping
// is disabled so shell commands used as step titles (e.g. "a && b") stay
// readable in the stored logs.
func MarshalStepLogs(steps []StepRecord) string {
	var sb strings.Builder

	enc := json.NewEncoder(&sb)
	enc.SetEscapeHTML(false)

	for _, step := range steps {
		if err := enc.Encode(step); err != nil {
			continue
		}
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// AddLogStep appends a step to the deployment logs in whichever format the
// stored blob uses: a structured record for NDJSON blobs, or a "[sk-step]"
// marker plus message lines for legacy text blobs (old deployments and
// GitHub-action builds).
func (d *Deployment) AddLogStep(step StepRecord) {
	if _, ok := ParseStepLogs(d.Logs.ValueOrZero()); ok || d.Logs.ValueOrZero() == "" {
		blob := d.Logs.ValueOrZero()

		if blob != "" {
			blob += "\n"
		}

		d.Logs.SetValid(blob + MarshalStepLogs([]StepRecord{step}))

		return
	}

	lines := []string{LogStep(step.Title)}

	if step.Message != "" {
		lines = append(lines, step.Message)
	}

	d.AddLogs(lines)
}

// prepareStepLogs renders structured step records into the log objects
// returned by the API. Durations come straight from the recorded
// start/finish timestamps, so ordering plays no role in the calculation.
func (d *Deployment) prepareStepLogs(steps []StepRecord, isStatusChecks bool) []*Log {
	logs := make([]*Log, 0, len(steps))

	for _, step := range steps {
		log := &Log{
			Title:   step.Title,
			Message: step.Message,
			Status:  step.Status != StepStatusFailed,
			Payload: step.Payload,
		}

		if step.Title == "deploy" && log.Message == "" {
			if d.Error.ValueOrZero() != "" {
				log.Message = d.Error.ValueOrZero()
				log.Status = false
			} else if d.UploadResult != nil {
				log.Message = d.UploadResult.message()
			}
		}

		finishedAt := step.FinishedAt

		if finishedAt == 0 && d.StoppedAt.Valid {
			finishedAt = d.StoppedAt.Unix() * 1000
		}

		if step.StartedAt > 0 && finishedAt > step.StartedAt {
			log.Duration = (finishedAt - step.StartedAt + 500) / 1000
		}

		logs = append(logs, log)
	}

	if isStatusChecks && len(logs) > 0 && d.StatusChecksPassed.Valid {
		logs[len(logs)-1].Status = d.StatusChecksPassed.Bool
	}

	// A stopped or crashed deployment kills the runner before it closes the
	// current step, leaving the last record without a status. Reconcile it
	// with the deployment outcome instead of rendering it as successful.
	if !isStatusChecks && len(steps) > 0 {
		last := steps[len(steps)-1]

		if last.FinishedAt == 0 && last.Status == "" && d.ExitCode.Valid && d.ExitCode.ValueOrZero() != 0 {
			lastLog := logs[len(logs)-1]
			lastLog.Status = false

			if d.ExitCode.ValueOrZero() == ExitCodeStopped {
				if lastLog.Message != "" {
					lastLog.Message += "\n"
				}

				lastLog.Message += "Deployment has been stopped manually."
			}
		}
	}

	return logs
}
