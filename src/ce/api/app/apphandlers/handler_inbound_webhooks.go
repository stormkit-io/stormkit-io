package apphandlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/dlclark/regexp2"
	"gopkg.in/guregu/null.v3"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/github"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

const typeCommit = "commit"
const typePullRequest = "pull_request"

// TriggerDeployInput represents the input for the TriggerDeploy function.
type TriggerDeployInput struct {
	Fail              bool   // used to debug leaving failure pr comments
	Repo              string // represents the base repository that the app is created
	CheckoutRepo      string // represents the repository that will be checked out
	IsFork            bool   // whether or not this deployment is a fork
	Branch            string
	Message           string
	EventType         string
	CommitSha         string
	PullRequestNumber int64

	payload any // The payload that is sent by the provider - we store this in the database.
}

// NewTriggerDeployInput is a helper function to
// initiate a new TriggerDeployInput instance.
func NewTriggerDeployInput(repo, branch string) TriggerDeployInput {
	return TriggerDeployInput{
		Repo:   repo,
		Branch: branch,
	}
}

func handlerInboundWebhooks(req *shttp.RequestContext) *shttp.Response {
	input, err := processMessage(req)

	// no-op
	if input == nil && err == nil {
		return shttp.NoContent()
	}

	if input == nil && err != nil {
		return shttp.Forbidden().SetError(err)
	}

	response := TriggerDeploy(req.Context(), *input)

	if response == nil {
		return shttp.NoContent()
	}

	if response.Error != nil && input.PullRequestNumber != 0 {
		slog.Errorf("error while auto deploying: %v", response.Error)
	}

	return response
}

func processMessage(req *shttp.RequestContext) (*TriggerDeployInput, error) {
	provider := req.Vars()["provider"]

	switch provider {
	case "github":
		return processGithubPayload(req)

	case "bitbucket":
		return processBitbucketPayload(req)

	case "gitlab":
		return processGitlabPayload(req)
	}

	return nil, nil
}

// TriggerDeploy triggers a new deploy given the repository, and the branch name.
// See tests for an example input event.
func TriggerDeploy(ctx context.Context, input TriggerDeployInput) *shttp.Response {
	// Do not deploy automatically sample projects
	if input.Repo == app.SampleProjectRepo {
		return nil
	}

	// This is mostly for GitLab as we may end up deploying the same commit again and
	// again because GitLab sends the same payload.
	if alreadyBuilt, err := commitHasBeenBuilt(ctx, input); err != nil || alreadyBuilt {
		if err != nil {
			return shttp.Error(err)
		}

		return &shttp.Response{
			Status: http.StatusAlreadyReported,
		}
	}

	apps, err := app.NewStore().DeployCandidates(ctx, input.Repo)

	if err != nil {
		return shttp.Error(err)
	}

	numberOfBuilds := 0

	for _, a := range FilterDeployCandidates(input, apps) {
		if input.IsFork {
			a.ShouldPublish = false
		}

		if a.EnvDefaultBranch != input.Branch {
			a.ShouldPublish = false
		}

		depl := deploy.New(a.App)
		depl.WebhookEvent = input.payload
		depl.Branch = input.Branch
		depl.Env = a.EnvName
		depl.EnvID = a.EnvID
		depl.IsAutoDeploy = true
		depl.Commit.ID = null.NewString(input.CommitSha, input.CommitSha != "")
		depl.CheckoutRepo = input.CheckoutRepo
		depl.IsFork = input.IsFork
		depl.BuildConfig = a.BuildConfig
		depl.ShouldPublish = a.ShouldPublish

		if a.SchemaConf != nil && a.SchemaConf.MigrationsEnabled {
			depl.MigrationsPath = null.StringFrom(a.SchemaConf.MigrationsPath)
		}

		if input.PullRequestNumber != 0 {
			depl.PullRequestNumber = null.NewInt(input.PullRequestNumber, true)
		}

		if err := deployservice.New().Deploy(ctx, a.App, depl); err != nil {
			isContextCanceled := errors.Is(err, context.Canceled)

			if !isContextCanceled {
				slog.Errorf("auto deployment failed for app id=%d, clone url:%s, err=%v", a.ID, depl.CheckoutRepo, err)
			}

			return shttp.Error(err)
		}

		// Post the status check if it's a github repo.
		cnf := admin.MustConfig()

		if a.IsGithub() && cnf.IsGithubEnabled() {
			err = github.CreateStatus(a.Repo, depl.Branch, cnf.DeploymentLogsURL(depl.AppID, depl.ID), github.StatusPending)

			if err != nil {
				slog.Errorf("error while updating github status: %s", err.Error())
			}
		}

		numberOfBuilds = numberOfBuilds + 1
	}

	if numberOfBuilds > 0 {
		return shttp.OK()
	}

	return shttp.NoContent()
}

// FilterDeployCandidates checks the following conditions and determines
// whether a deploy candidate should be deployed or not.
//
//  1. If the branch name matches the branch of an environment, return
//     that environment
//  2. If we still have nothing, check the Auto Deploy Branch config. Return
//     all matches. If nothing is found, return empty.
func FilterDeployCandidates(input TriggerDeployInput, dcs []*app.DeployCandidate) []*app.DeployCandidate {
	filtered := []*app.DeployCandidate{}

	// All candidates have auto_deploy turned on
	for _, dc := range dcs {
		patternBranches := dc.AutoDeployBranches.ValueOrZero()
		patternCommits := dc.AutoDeployCommits.ValueOrZero()

		// If the pattern is empty, it means we want to deploy all branches/commits
		if patternBranches == "" && patternCommits == "" {
			filtered = append(filtered, dc)
			continue
		}

		if patternBranches != "" {
			// When the default branch is the same with the current branch include the dc
			// This feature is only available when deploy branches is specified
			if strings.EqualFold(dc.EnvDefaultBranch, input.Branch) {
				filtered = append(filtered, dc)
			} else if MatchPattern(patternBranches, input.Branch) {
				filtered = append(filtered, dc)
			}

			continue
		}

		// Make sure to build commits on the release branch only.
		if input.Branch != dc.EnvDefaultBranch {
			continue
		}

		if patternCommits != "" && input.Message != "" && MatchPattern(patternCommits, input.Message) {
			filtered = append(filtered, dc)
		}
	}

	return filtered
}

// commitHasBeenBuilt checks whether there is already a build for the commit or not.
func commitHasBeenBuilt(ctx context.Context, input TriggerDeployInput) (bool, error) {
	return deploy.NewStore().IsDeploymentAlreadyBuilt(ctx, input.CommitSha)
}

// MatchPattern matches the given branch name against the given glob pattern.
func MatchPattern(pattern, branch string) bool {
	r, err := regexp2.Compile(pattern, regexp2.IgnoreCase)

	if err != nil {
		return false
	}

	matched, err := r.MatchString(branch)

	if err != nil {
		slog.Errorf("error while matching string: %s", err.Error())
		return false
	}

	return matched
}
