package runner

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stretchr/testify/suite"
)

type StepLoggerSuite struct {
	suite.Suite
}

func (s *StepLoggerSuite) parse(reporter *ReporterModel) []deploy.StepRecord {
	steps, ok := deploy.ParseStepLogs(reporter.Logs())

	s.Require().True(ok)

	return steps
}

func (s *StepLoggerSuite) Test_OutputIsAttributedToCurrentStep() {
	reporter := NewReporter("")

	// The writer is captured once and reused across steps, like the
	// installer does: output must land in whichever step is current.
	file := reporter.File()

	reporter.AddStep("npm install")
	_, err := file.Write([]byte("added 12 packages\n"))
	s.NoError(err)

	reporter.AddStep("npm run build")
	_, err = file.Write([]byte("compiled\n"))
	s.NoError(err)

	steps := s.parse(reporter)

	s.Len(steps, 2)
	s.Equal("npm install", steps[0].Title)
	s.Equal("added 12 packages\n", steps[0].Message)
	s.Equal("npm run build", steps[1].Title)
	s.Equal("compiled\n", steps[1].Message)

	// The first step was closed by the second one.
	s.Equal(deploy.StepStatusSuccess, steps[0].Status)
	s.NotZero(steps[0].FinishedAt)
	s.GreaterOrEqual(steps[0].FinishedAt, steps[0].StartedAt)

	// The second step is still running.
	s.Zero(steps[1].FinishedAt)
}

func (s *StepLoggerSuite) Test_OutputBeforeFirstStepIsDropped() {
	reporter := NewReporter("")

	_, err := reporter.File().Write([]byte("boot noise\n"))
	s.NoError(err)

	reporter.AddStep("checkout main")

	steps := s.parse(reporter)

	s.Len(steps, 1)
	s.Empty(steps[0].Message)
}

func (s *StepLoggerSuite) Test_SystemMarkersCloseWithoutOpening() {
	reporter := NewReporter("")

	reporter.AddStep("saving build cache")
	reporter.AddLine("saved build cache (193MB)")
	reporter.AddStep("[system] building finished")

	steps := s.parse(reporter)

	s.Len(steps, 1)
	s.Equal("saving build cache", steps[0].Title)
	s.Equal("saved build cache (193MB)\n", steps[0].Message)
	s.NotZero(steps[0].FinishedAt)
}

func (s *StepLoggerSuite) Test_CloseStep_MarksFailure() {
	reporter := NewReporter("")

	reporter.AddStep("npm run e2e")
	reporter.CloseStep(false)

	// A later close must not overwrite the recorded outcome.
	reporter.CloseStep(true)
	reporter.AddStep("[system] deployment finished")

	steps := s.parse(reporter)

	s.Len(steps, 1)
	s.Equal(deploy.StepStatusFailed, steps[0].Status)
}

func TestStepLoggerSuite(t *testing.T) {
	suite.Run(t, new(StepLoggerSuite))
}
