package apphandlers

import (
	"fmt"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"gopkg.in/go-playground/webhooks.v5/bitbucket"
)

// processBitbucketPayload processes a bitbucket payload and starts a new deployment.
func processBitbucketPayload(req *shttp.RequestContext) (*TriggerDeployInput, error) {
	hook, _ := bitbucket.New()
	payload, err := hook.Parse(
		req.Request,
		bitbucket.RepoPushEvent,
		bitbucket.PullRequestCreatedEvent,
		bitbucket.PullRequestMergedEvent,
	)

	if err != nil {
		event_type := req.Header.Get("X-Event-Key")
		slog.Infof("Bitbucket request failed for event type %s", event_type)

		return nil, err
	}

	input := TriggerDeployInput{
		payload: payload,
	}

	switch event := payload.(type) {

	// Commit event
	case bitbucket.RepoPushPayload:
		input.Branch = event.Push.Changes[0].New.Name
		input.Repo = fmt.Sprintf("bitbucket/%s", event.Repository.FullName)
		input.CheckoutRepo = fmt.Sprintf("bitbucket/%s", event.Repository.FullName)
		input.Message = event.Push.Changes[0].New.Target.Message
		input.EventType = event.Push.Changes[0].New.Target.Type // This value is either commit or something else. We don't care about the 'something else' case.
		input.IsFork = false

	// Pull request create event
	// Build the source branch in this case.
	case bitbucket.PullRequestCreatedPayload:
		input.Branch = event.PullRequest.Source.Branch.Name
		input.Repo = fmt.Sprintf("bitbucket/%s", event.Repository.FullName)
		input.CheckoutRepo = fmt.Sprintf("bitbucket/%s", event.PullRequest.Source.Repository.FullName)
		input.Message = event.PullRequest.Title
		input.EventType = typePullRequest
		input.PullRequestNumber = event.PullRequest.ID
		input.IsFork = !strings.EqualFold(input.CheckoutRepo, input.Repo)

	// Pull request merged event
	case bitbucket.PullRequestMergedPayload:
		input.Branch = event.PullRequest.Destination.Branch.Name
		input.Repo = fmt.Sprintf("bitbucket/%s", event.Repository.FullName)
		input.CheckoutRepo = input.Repo
		input.Message = event.PullRequest.Title
		input.EventType = typePullRequest
		input.IsFork = false

	default:
		return nil, nil
	}

	return &input, nil
}
