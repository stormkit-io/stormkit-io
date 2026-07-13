package deployhooks

import (
	"context"
	"fmt"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/bitbucket"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/github"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/gitlab"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

// pullRequestPreview automatically posts the preview link to the pull request screen.
func pullRequestPreview(details *AppDetails, d *deploy.Deployment) {
	previewLinksEnabled := true

	if d.BuildConfig != nil && d.BuildConfig.PreviewLinks.Valid {
		previewLinksEnabled = d.BuildConfig.PreviewLinks.ValueOrZero()
	}

	// Do not proceed if we don't have any pull request number.
	if details.PullRequestNumber == 0 || !previewLinksEnabled {
		return
	}

	var body string
	cnf := admin.MustConfig()

	switch d.ExitCode.ValueOrZero() {
	case 0:
		body =
			"#### Deployment completed\n\n" +
				"This pull request was successfully built by **[Stormkit](https://www.stormkit.io)**. You can preview it using the following link.\n" +
				fmt.Sprintf("> %s", cnf.PreviewURL(details.DisplayName, d.ID.String()))
	default:
		body =
			"#### Deployment failed\n\n" +
				"This pull request failed while building automatically on **[Stormkit](https://www.stormkit.io)**. " +
				"You can preview the logs using the following link.\n" +
				cnf.DeploymentLogsURL(d.AppID, d.ID)
	}

	if strings.HasPrefix(details.Repo, "github/") {
		PullRequestPreviewGithub(details, body)
	} else if strings.HasPrefix(details.Repo, "gitlab") {
		PullRequestPreviewGitlab(details, body)
	} else if strings.HasPrefix(details.Repo, "bitbucket") {
		PullRequestPreviewBitbucket(details, body)
	}
}

// PullRequestPreviewGithub creates a pull request preview for Github projects.
var PullRequestPreviewGithub = func(details *AppDetails, body string) {
	client, err := github.NewApp(details.Repo)

	if err != nil {
		slog.Errorf("failed while creating github client: %v", err)
		return
	}

	comment, _, err := client.Issues.CreateComment(
		context.Background(),
		client.Owner,
		client.Repo,
		int(details.PullRequestNumber),
		&github.IssueComment{
			Body: &body,
		})

	if err != nil {
		slog.Errorf("error while github creating pr comment: %v", err)
	}

	// Cleanup stuff for test environments.
	if config.IsTest() {
		if err == nil {
			_, err = client.Issues.DeleteComment(context.Background(), client.Owner, client.Repo, *comment.ID)
		}

		if err != nil {
			panic(err)
		}
	}
}

// pullRequestPreviewGitlab creates a pull request preview for Gitlab projects.
func PullRequestPreviewGitlab(details *AppDetails, body string) {
	client, err := gitlab.NewClient(details.UserID)

	if err != nil {
		slog.Errorf("failed while creating bitbucket client: %v", err)
		return
	}

	pid := client.SanitizeRepo(details.Repo)
	mid := int(details.PullRequestNumber)
	mrOpts := &gitlab.CreateMergeRequestDiscussionOptions{Body: &body}
	comment, _, err := client.Discussions.CreateMergeRequestDiscussion(pid, mid, mrOpts)

	if err == nil && comment != nil && !strings.Contains(body, "Deployment failed") {
		resolved := true
		rsOpts := &gitlab.ResolveMergeRequestDiscussionOptions{Resolved: &resolved}

		if _, _, err := client.Discussions.ResolveMergeRequestDiscussion(pid, mid, comment.ID, rsOpts); err != nil {
			slog.Errorf("merge request resolve: %v", err)
		}
	}

	if err != nil {
		slog.Errorf("failed creating a gitlab preview comment: %v, app=%d, pr=%d", err, details.AppID, details.PullRequestNumber)
	}

	// Cleanup stuff for test environments.
	if config.IsTest() {
		if err == nil {
			_, err = client.Discussions.DeleteMergeRequestDiscussionNote(
				client.SanitizeRepo(details.Repo),
				int(details.PullRequestNumber),
				comment.ID,
				comment.Notes[0].ID,
			)
		}

		if err != nil {
			panic(err)
		}
	}
}

// pullRequestPreviewBitbucket creates a pull request preview for Bitbucket projects.
func PullRequestPreviewBitbucket(details *AppDetails, body string) {
	client, err := bitbucket.NewClientWithScope(details.UserID, []string{
		bitbucket.PermissionRepositoryWrite,
	})

	if err != nil {
		slog.Errorf("failed while creating bitbucket client: %v", err)
		return
	}

	app := &bitbucket.App{Repo: details.Repo}

	comment, err := client.PullRequestComment(app, body, details.PullRequestNumber)

	if config.IsTest() {
		if err == nil {
			err = client.PullRequestRemoveComment(app, details.PullRequestNumber, comment.ID)
		}

		if err != nil {
			panic(err)
		}
	}
}
