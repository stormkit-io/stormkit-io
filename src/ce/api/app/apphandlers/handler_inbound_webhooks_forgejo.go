package apphandlers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type forgejoRepository struct {
	FullName      string `json:"full_name"`
	DefaultBranch string `json:"default_branch"`
}

type forgejoPushPayload struct {
	Ref          string `json:"ref"`
	TotalCommits int    `json:"total_commits"`
	Commits      []struct {
		Message  string   `json:"message"`
		Added    []string `json:"added"`
		Removed  []string `json:"removed"`
		Modified []string `json:"modified"`
	} `json:"commits"`
	Repository forgejoRepository `json:"repository"`
}

type forgejoPullRequestPayload struct {
	Action      string `json:"action"`
	Number      int64  `json:"number"`
	PullRequest struct {
		Title string `json:"title"`
		Head  struct {
			Ref  string            `json:"ref"`
			Sha  string            `json:"sha"`
			Repo forgejoRepository `json:"repo"`
		} `json:"head"`
	} `json:"pull_request"`
	Repository forgejoRepository `json:"repository"`
}

// forgejoPRActions are the pull request actions that carry new code to build.
var forgejoPRActions = []string{"opened", "reopened", "synchronized"}

// processForgejoPayload parses Forgejo (and Gitea) webhook payloads.
func processForgejoPayload(req *shttp.RequestContext) (*TriggerDeployInput, error) {
	event := utils.GetString(req.Header.Get("X-Forgejo-Event"), req.Header.Get("X-Gitea-Event"))

	switch event {
	case "push":
		payload := forgejoPushPayload{}

		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			return nil, err
		}

		// Pushes without commits, such as deleting a branch, have nothing new to build.
		if len(payload.Commits) == 0 {
			return nil, nil
		}

		input := &TriggerDeployInput{
			payload:         payload,
			Repo:            fmt.Sprintf("forgejo/%s", payload.Repository.FullName),
			Branch:          strings.TrimPrefix(payload.Ref, "refs/heads/"),
			Message:         strings.Split(payload.Commits[0].Message, "\n")[0],
			EventType:       typeCommit,
			ChangesComplete: len(payload.Commits) == payload.TotalCommits,
		}

		input.CheckoutRepo = input.Repo

		for _, commit := range payload.Commits {
			input.ChangedFiles = append(input.ChangedFiles, commit.Added...)
			input.ChangedFiles = append(input.ChangedFiles, commit.Modified...)
			input.ChangedFiles = append(input.ChangedFiles, commit.Removed...)
		}

		// Same as GitLab: other branches are built through their pull requests.
		if !strings.EqualFold(input.Branch, payload.Repository.DefaultBranch) {
			return nil, nil
		}

		return input, nil

	case "pull_request":
		payload := forgejoPullRequestPayload{}

		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			return nil, err
		}

		if !utils.InSliceString(forgejoPRActions, payload.Action) {
			return nil, nil
		}

		input := &TriggerDeployInput{
			payload:           payload,
			Repo:              fmt.Sprintf("forgejo/%s", payload.Repository.FullName),
			CheckoutRepo:      fmt.Sprintf("forgejo/%s", payload.PullRequest.Head.Repo.FullName),
			Branch:            payload.PullRequest.Head.Ref,
			CommitSha:         payload.PullRequest.Head.Sha,
			Message:           payload.PullRequest.Title,
			PullRequestNumber: payload.Number,
			EventType:         typePullRequest,
		}

		input.IsFork = !strings.EqualFold(input.CheckoutRepo, input.Repo)

		return input, nil
	}

	return nil, nil
}
