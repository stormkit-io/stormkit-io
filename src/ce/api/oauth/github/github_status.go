package github

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/go-github/v71/github"
)

// CreateStatus creates a new status
func CreateStatus(repo, branch, url, status string) error {
	gh, err := NewApp(repo)

	if err != nil || gh == nil {
		return err
	}

	ctx := context.Background()

	// Grab the latest commit for the branch
	ghBranch, _, err := gh.Repositories.GetBranch(ctx, gh.Owner, gh.Repo, branch, 0)

	if err != nil || ghBranch == nil || ghBranch.Commit == nil {
		return err
	}

	sha := ghBranch.Commit.SHA

	if sha == nil {
		return nil
	}

	var text string

	if status == StatusFailure {
		text = "Deployment failed"
	} else if status == StatusSuccess {
		text = "Deployment completed"
	} else {
		text = "Deploying application"
	}

	_, _, err = gh.Repositories.CreateStatus(
		ctx,
		gh.Owner,
		gh.Repo,
		*sha,
		&github.RepoStatus{
			Description: aws.String(text),
			TargetURL:   aws.String(url),
			State:       aws.String(status),
			Context:     aws.String("Stormkit"),
		})

	return err
}
