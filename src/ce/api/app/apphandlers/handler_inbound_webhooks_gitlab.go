package apphandlers

import (
	"fmt"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"gopkg.in/go-playground/webhooks.v5/gitlab"
)

const VisibilityLevelPublic = 20

func processGitlabPayload(req *shttp.RequestContext) (*TriggerDeployInput, error) {
	hook, _ := gitlab.New()
	payload, err := hook.Parse(
		req.Request,
		gitlab.PushEvents,
		gitlab.MergeRequestEvents,
		gitlab.CommentEvents,
	)

	if err != nil {
		event_type := req.Header.Get("X-Gitlab-Event")
		slog.Infof("Gitlab Request failed for event type %s", event_type)
		return nil, err
	}

	input := TriggerDeployInput{
		payload: payload,
	}

	switch event := payload.(type) {

	// Commit event
	case gitlab.PushEventPayload:
		input.Branch = strings.Replace(event.Ref, "refs/heads/", "", 1)
		input.Repo = fmt.Sprintf("gitlab/%s", event.Project.PathWithNamespace)
		input.CheckoutRepo = input.Repo
		input.EventType = typeCommit
		input.Message = strings.Split(event.Commits[0].Message, "\n")[0]
		input.IsFork = false

		// Do not build commits that were not in default branch because:
		// 1. If the commit is made into a pull request - we'll receive the event anyways.
		// 2. If the commit is made outside of a pull request, we don't have anywhere to report anyways.
		if !strings.EqualFold(input.Branch, event.Project.DefaultBranch) {
			return nil, nil
		}

	// Pull request event
	// Build the source branch in this case.
	//
	// Known issue:
	// - GitLab sends a second push event when the build is complete and we leave a message on the PR.
	//   But we overcome this by checking the commits that were built previously and we don't rebuild.
	case gitlab.MergeRequestEventPayload:
		// This is a no-op, we only want to build when the pull request is opened.
		// When there is a new commit on the PR, we still receive `opened` state anyways.
		if event.ObjectAttributes.State != "opened" {
			return nil, nil
		}

		input.Repo = fmt.Sprintf("gitlab/%s", event.Project.PathWithNamespace)
		input.IsFork = strings.Compare(input.CheckoutRepo, input.Repo) != 0
		input.CheckoutRepo = fmt.Sprintf("gitlab/%s", event.ObjectAttributes.Source.PathWithNamespace)
		input.Message = strings.Split(event.ObjectAttributes.LastCommit.Message, "\n")[0]
		input.PullRequestNumber = event.ObjectAttributes.IID
		input.Branch = event.ObjectAttributes.SourceBranch
		input.CommitSha = event.ObjectAttributes.LastCommit.ID
		input.EventType = typePullRequest

	default:
		return nil, nil
	}

	return &input, nil
}
