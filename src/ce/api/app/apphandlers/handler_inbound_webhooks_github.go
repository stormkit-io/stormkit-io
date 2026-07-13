package apphandlers

import (
	"fmt"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"gopkg.in/go-playground/webhooks.v5/github"
)

var whiteList = []string{
	string(github.PushEvent),
	string(github.PullRequestEvent),
	string(github.IssueCommentEvent),
	string(github.CheckSuiteEvent),
}

// processGithubPayload processes a github payload and starts a new deployment.
func processGithubPayload(req *shttp.RequestContext) (*TriggerDeployInput, error) {
	hook, _ := github.New()
	eventType := req.Header.Get("X-GitHub-Event")

	// Not sure why we receive this hook but it's not required for this endpoint
	if !utils.InSliceString(whiteList, eventType) {
		return nil, nil
	}

	payload, err := hook.Parse(
		req.Request,
		github.PushEvent,
		github.PullRequestEvent,
		github.IssueCommentEvent,
		// until we get answer from github
		// we will parse this and no-op
		github.CheckSuiteEvent,
	)

	if err != nil {
		slog.Errorf("Github request failed for event type=%s, err=%s", eventType, err.Error())
		return nil, nil
	}

	input := TriggerDeployInput{
		payload: payload,
	}

	switch event := payload.(type) {

	// Commit event
	case github.PushPayload:
		input.Branch = strings.Replace(event.Ref, "refs/heads/", "", 1)
		input.Message = event.HeadCommit.Message
		input.Repo = fmt.Sprintf("github/%s", event.Repository.FullName)
		input.CheckoutRepo = fmt.Sprintf("github/%s", event.Repository.FullName)
		input.EventType = typeCommit
		input.IsFork = false

		// Pushed something else: for instance a tag.
		if input.Message == "" {
			return nil, nil
		}

	// Pull request event
	case github.PullRequestPayload:
		input.Message = event.PullRequest.Title
		input.EventType = typePullRequest
		input.Repo = fmt.Sprintf("github/%s", event.Repository.FullName)
		input.CheckoutRepo = fmt.Sprintf("github/%s", event.PullRequest.Head.Repo.FullName)
		input.IsFork = event.PullRequest.Head.Repo.FullName != event.PullRequest.Base.Repo.FullName
		input.PullRequestNumber = event.PullRequest.Number
		input.Branch = event.PullRequest.Head.Ref

		if event.Action != "opened" && event.Action != "synchronize" {
			return nil, nil
		}

	default:
		return nil, nil
	}

	if strings.Contains(input.Branch, "refs/tags/") {
		return nil, nil
	}

	return &input, nil
}
