package deployhooks

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/github"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/discord"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

var StatusChecksEnabled = true

// Exec executes the deployment hooks once a deployment is completed.
var Exec = func(ctx context.Context, d *deploy.Deployment) {
	// TODO: See if you can remove this call. It seems like most of the fields
	// that we are requesting are already in the `d` object.
	details, err := NewStore().AppDetailsForHooks(d.ID)

	if err != nil {
		slog.Errorf("failed while fetching details for provider: %v", err)
		return
	}

	if details == nil {
		return
	}

	whs := app.NewStore().OutboundWebhooks(ctx, d.AppID)
	status := "success"

	if d.ExitCode.ValueOrZero() != 0 {
		status = "failed"
	}

	cnf := admin.MustConfig()
	args := app.OutboundWebhookSettings{
		AppID:                  d.AppID,
		DeploymentID:           d.ID,
		DeploymentStatus:       status,
		EnvironmentName:        d.Env,
		DeploymentError:        d.Error.ValueOrZero(),
		DeploymentLogsEndpoint: cnf.DeploymentLogsURL(d.AppID, d.ID),
		DeploymentEndpoint:     cnf.PreviewURL(d.DisplayName, d.ID.String()),
	}

	for _, wh := range whs {
		if status == "success" && wh.TriggerOnDeploySuccess() {
			wh.Dispatch(args)
		}

		if status == "failed" && wh.TriggerOnDeployFailed() {
			wh.Dispatch(args)
		}
	}

	if StatusChecksEnabled {
		statusChecks(details, d)
	}

	pullRequestPreview(details, d)

	c := config.Get()

	// Notify deployment completed only when it's successful
	if d.ExitCode.ValueOrZero() == 0 && d.Error.ValueOrZero() == "" {
		notifyDeploymentCompleted(c.Reporting.DiscordDeploymentsSuccessChannel, d)
	} else {
		notifyDeploymentCompleted(c.Reporting.DiscordDeploymentsFailedChannel, d)
	}
}

// statusChecks creates a status check for the deployment with a finite
// status like success or failure.
func statusChecks(details *AppDetails, d *deploy.Deployment) {
	if !strings.HasPrefix(details.Repo, "github/") || !admin.MustConfig().IsGithubEnabled() {
		return
	}

	var status string

	if d.ExitCode.Valid && d.ExitCode.ValueOrZero() == 0 {
		status = github.StatusSuccess
	} else {
		status = github.StatusFailure
	}

	cnf := admin.MustConfig()
	err := github.CreateStatus(details.Repo, d.Branch, cnf.DeploymentLogsURL(d.AppID, d.ID), status)

	if err != nil {
		message := err.Error()
		ignoredErrors := []string{"403", "404", "422", "not accessible"}
		shouldPrint := true

		for _, ignoredError := range ignoredErrors {
			if strings.Contains(message, ignoredError) {
				shouldPrint = false
				break
			}
		}

		if shouldPrint {
			slog.Errorf("error while creating github status: %v", err)
		}
	}
}

// notifies discord channel that a deployment has ended.
func notifyDeploymentCompleted(channelHook string, d *deploy.Deployment) {
	cnf := admin.MustConfig()
	did := d.ID.String()

	go discord.Notify(channelHook, discord.Payload{
		Embeds: []discord.PayloadEmbed{
			{
				Title:     fmt.Sprintf("Deployment Completed: %d", d.ID),
				Timestamp: time.Now().Format(time.RFC3339),
				URL:       cnf.PreviewURL(d.DisplayName, did),
				Fields: []discord.PayloadField{
					{
						Name:  "App ID",
						Value: fmt.Sprintf("%d", d.AppID),
					},
					{
						Name:  "Endpoint",
						Value: cnf.PreviewURL(d.DisplayName, did),
					},
					{
						Name:  "Exit code",
						Value: fmt.Sprintf("%d", d.ExitCode.ValueOrZero()),
					},
					{
						Name:  "Deployment Logs URL",
						Value: cnf.DeploymentLogsURL(d.AppID, d.ID),
					},
					{
						Name:  "Is Auto Deploy?",
						Value: strconv.FormatBool(d.IsAutoDeploy),
					},
				}},
		},
	})
}

var Publish = deploy.Publish
