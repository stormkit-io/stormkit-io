package deploy_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type DeploymentLogsSuite struct {
	suite.Suite
}

func (s *DeploymentLogsSuite) Test_MarshalAndParse_RoundTrip() {
	steps := []deploy.StepRecord{
		{Title: "npm install", Message: "added 12 packages\n", StartedAt: 1000, FinishedAt: 5000, Status: deploy.StepStatusSuccess},
		{Title: "npm run build && echo done", StartedAt: 5000, FinishedAt: 9000, Status: deploy.StepStatusSuccess},
	}

	blob := deploy.MarshalStepLogs(steps)

	// Shell commands used as titles must stay readable in the stored blob.
	s.Contains(blob, `"title":"npm run build && echo done"`)

	parsed, ok := deploy.ParseStepLogs(blob)

	s.True(ok)
	s.Equal(steps, parsed)
}

func (s *DeploymentLogsSuite) Test_ParseStepLogs_RejectsLegacyText() {
	_, ok := deploy.ParseStepLogs("[sk-step] npm install [ts:1726053541]\nadded 12 packages")

	s.False(ok)
}

func (s *DeploymentLogsSuite) Test_ParseStepLogs_SkipsPartialTrailingLine() {
	blob := deploy.MarshalStepLogs([]deploy.StepRecord{
		{Title: "npm install", StartedAt: 1000, FinishedAt: 2000},
	}) + "\n{\"title\":\"npm run bui"

	parsed, ok := deploy.ParseStepLogs(blob)

	s.True(ok)
	s.Len(parsed, 1)
	s.Equal("npm install", parsed[0].Title)
}

func (s *DeploymentLogsSuite) Test_PrepareLogs_StructuredSteps() {
	d := &deploy.Deployment{
		ExitCode: null.IntFrom(0),
		UploadResult: &deploy.UploadResult{
			ServerBytes: 591919,
		},
		Logs: null.StringFrom(deploy.MarshalStepLogs([]deploy.StepRecord{
			{Title: "npm run build", Message: "compiled\n", StartedAt: 1_726_053_541_000, FinishedAt: 1_726_053_711_000, Status: deploy.StepStatusSuccess},
			{Title: "saving build cache", Message: "saved build cache (193MB)\n", StartedAt: 1_726_053_711_000, FinishedAt: 1_726_053_756_000, Status: deploy.StepStatusSuccess},
			{Title: "deploy", StartedAt: 1_726_053_756_000, FinishedAt: 1_726_053_767_400, Status: deploy.StepStatusSuccess},
			{Title: "database migrations", Message: "No new migrations to apply.", StartedAt: 1_726_053_767_500, FinishedAt: 1_726_053_768_500, Status: deploy.StepStatusSuccess},
		})),
	}

	b, err := json.Marshal(d.PrepareLogs(d.Logs.ValueOrZero(), false))
	s.Nil(err)
	s.JSONEq(`[
		{
			"title": "npm run build",
			"duration": 170,
			"message": "compiled\n",
			"status": true,
			"payload": null
		},
		{
			"title": "saving build cache",
			"duration": 45,
			"message": "saved build cache (193MB)\n",
			"status": true,
			"payload": null
		},
		{
			"title": "deploy",
			"duration": 11,
			"message": "\nSuccessfully deployed server side.\nPackage size: 591.9kB\n\n",
			"status": true,
			"payload": null
		},
		{
			"title": "database migrations",
			"duration": 1,
			"message": "No new migrations to apply.",
			"status": true,
			"payload": null
		}
	]`, string(b))
}

func (s *DeploymentLogsSuite) Test_PrepareLogs_StructuredSteps_FailedStep() {
	d := &deploy.Deployment{
		ExitCode: null.IntFrom(1),
		Logs: null.StringFrom(deploy.MarshalStepLogs([]deploy.StepRecord{
			{Title: "npm install", StartedAt: 1000, FinishedAt: 2000, Status: deploy.StepStatusSuccess},
			{Title: "npm run build", Message: "type error\n", StartedAt: 2000, FinishedAt: 9000, Status: deploy.StepStatusFailed},
		})),
	}

	logs := d.PrepareLogs(d.Logs.ValueOrZero(), false)

	s.Len(logs, 2)
	s.True(logs[0].Status)
	s.False(logs[1].Status)
}

func (s *DeploymentLogsSuite) Test_PrepareLogs_StructuredSteps_DeployError() {
	d := &deploy.Deployment{
		Error: null.StringFrom("Error: cannot upload artifacts"),
		Logs: null.StringFrom(deploy.MarshalStepLogs([]deploy.StepRecord{
			{Title: "deploy", StartedAt: 1000, Status: deploy.StepStatusSuccess},
		})),
	}

	logs := d.PrepareLogs(d.Logs.ValueOrZero(), false)

	s.Len(logs, 1)
	s.Equal("Error: cannot upload artifacts", logs[0].Message)
	s.False(logs[0].Status)
}

func (s *DeploymentLogsSuite) Test_PrepareLogs_StructuredSteps_RunningStep() {
	stoppedAt := utils.Unix{Time: time.Unix(100, 0), Valid: true}

	d := &deploy.Deployment{
		StoppedAt: stoppedAt,
		Logs: null.StringFrom(deploy.MarshalStepLogs([]deploy.StepRecord{
			{Title: "npm run build", StartedAt: 88_000},
		})),
	}

	logs := d.PrepareLogs(d.Logs.ValueOrZero(), false)

	s.Len(logs, 1)
	// The step never recorded a finish; the deployment's stop time caps it.
	s.Equal(int64(12), logs[0].Duration)
	s.True(logs[0].Status)
}

func (s *DeploymentLogsSuite) Test_PrepareLogs_StructuredSteps_StoppedDeployment() {
	d := &deploy.Deployment{
		ExitCode:  null.IntFrom(deploy.ExitCodeStopped),
		StoppedAt: utils.Unix{Time: time.Unix(100, 0), Valid: true},
		Logs: null.StringFrom(deploy.MarshalStepLogs([]deploy.StepRecord{
			{Title: "npm install", StartedAt: 80_000, FinishedAt: 88_000, Status: deploy.StepStatusSuccess},
			// The runner was killed by the stop before closing this step.
			{Title: "npm run build", StartedAt: 88_000},
		})),
	}

	logs := d.PrepareLogs(d.Logs.ValueOrZero(), false)

	s.Len(logs, 2)
	s.True(logs[0].Status)
	s.False(logs[1].Status)
	s.Contains(logs[1].Message, "Deployment has been stopped manually.")
}

func (s *DeploymentLogsSuite) Test_ParseStepLogs_SkipsUntitledRecords() {
	blob := deploy.MarshalStepLogs([]deploy.StepRecord{
		{Title: "npm install", StartedAt: 1000, FinishedAt: 2000},
	}) + "\n{\"level\":\"info\",\"msg\":\"json build output without a title\"}"

	parsed, ok := deploy.ParseStepLogs(blob)

	s.True(ok)
	s.Len(parsed, 1)
	s.Equal("npm install", parsed[0].Title)
}

func (s *DeploymentLogsSuite) Test_PrepareLogs_StructuredSteps_StatusChecks() {
	d := &deploy.Deployment{
		StatusChecksPassed: null.BoolFrom(false),
		StatusChecks: null.StringFrom(deploy.MarshalStepLogs([]deploy.StepRecord{
			{Title: "npm run e2e", StartedAt: 1000, FinishedAt: 2000, Status: deploy.StepStatusSuccess},
		})),
	}

	logs := d.PrepareLogs(d.StatusChecks.ValueOrZero(), true)

	s.Len(logs, 1)
	s.False(logs[0].Status)
}

func (s *DeploymentLogsSuite) Test_AddLogStep_AppendsToStructuredBlob() {
	d := &deploy.Deployment{
		Logs: null.StringFrom(deploy.MarshalStepLogs([]deploy.StepRecord{
			{Title: "deploy", StartedAt: 1000, FinishedAt: 2000, Status: deploy.StepStatusSuccess},
		})),
	}

	d.AddLogStep(deploy.StepRecord{
		Title:      "database migrations",
		Message:    "No new migrations to apply.",
		StartedAt:  3000,
		FinishedAt: 4000,
		Status:     deploy.StepStatusSuccess,
	})

	steps, ok := deploy.ParseStepLogs(d.Logs.ValueOrZero())

	s.True(ok)
	s.Len(steps, 2)
	s.Equal("database migrations", steps[1].Title)
	s.Equal(int64(3000), steps[1].StartedAt)
}

func (s *DeploymentLogsSuite) Test_AddLogStep_FallsBackToLegacyText() {
	d := &deploy.Deployment{
		Logs: null.StringFrom("[sk-step] npm install [ts:1726053541]\nadded 12 packages"),
	}

	d.AddLogStep(deploy.StepRecord{
		Title:   "database migrations",
		Message: "No new migrations to apply.",
	})

	logs := d.Logs.ValueOrZero()

	s.Contains(logs, "[sk-step] database migrations [ts:")
	s.Contains(logs, "No new migrations to apply.")
	s.NotContains(logs, `"title"`)
}

func TestDeploymentLogsSuite(t *testing.T) {
	suite.Run(t, new(DeploymentLogsSuite))
}
