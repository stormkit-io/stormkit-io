package apphandlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"runtime/debug"
	"strings"
	"time"

	"github.com/dlclark/regexp2"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/github"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// ErrInvalidWebhookSecret is returned when an inbound webhook cannot be verified.
var ErrInvalidWebhookSecret = errors.New("invalid webhook secret")

// errWebhookAppLookup wraps failures to load the app a webhook secret points
// to. These are server errors, not authentication failures, so providers retry
// the delivery instead of treating the hook as misconfigured.
var errWebhookAppLookup = errors.New("cannot load the webhook's app")

// maxWebhookPayloadSize is the largest webhook payload accepted. It matches
// GitHub's limit, the largest of the supported providers.
const maxWebhookPayloadSize = 25 << 20

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
	ChangedFiles      []string
	ChangesComplete   bool

	// AppID limits the deployment to a single app when the webhook was
	// verified with a per-app secret. Zero means every app on the repo.
	AppID types.ID

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
	req.Body = http.MaxBytesReader(nil, req.Body, maxWebhookPayloadSize)

	input, err := processMessage(req)

	// no-op
	if input == nil && err == nil {
		return shttp.NoContent()
	}

	if errors.Is(err, errWebhookAppLookup) {
		return shttp.Error(err)
	}

	if err != nil {
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
	v := webhookVerifier{req: req}

	switch req.Vars()["provider"] {
	case "github":
		return processGithubPayload(req)

	case "bitbucket":
		return v.appPayload(processBitbucketPayload)

	case "gitlab":
		return v.appPayload(processGitlabPayload)
	}

	return nil, nil
}

// webhookVerifier authenticates inbound webhooks before they trigger deployments.
type webhookVerifier struct {
	req *shttp.RequestContext
}

// appPayload verifies the per-app secret in the webhook URL before parsing the
// payload with parse, so requests without a valid secret are rejected without
// reading their body. The payload must belong to the app's repository.
func (v webhookVerifier) appPayload(parse func(*shttp.RequestContext) (*TriggerDeployInput, error)) (*TriggerDeployInput, error) {
	verified, err := v.app()

	if err != nil {
		return nil, err
	}

	input, err := parse(v.req)

	if input == nil || err != nil {
		return input, err
	}

	if !strings.EqualFold(verified.Repo, input.Repo) {
		return nil, ErrInvalidWebhookSecret
	}

	input.AppID = verified.ID

	return input, nil
}

// app returns the app whose secret the webhook URL carries.
func (v webhookVerifier) app() (*app.App, error) {
	appID, err := utils.DecryptID(v.req.Vars()["secret-id"])

	if err != nil || appID == 0 {
		return nil, ErrInvalidWebhookSecret
	}

	a, err := app.NewStore().AppByID(v.req.Context(), appID)

	if err != nil {
		return nil, fmt.Errorf("%w: %w", errWebhookAppLookup, err)
	}

	if a == nil {
		return nil, ErrInvalidWebhookSecret
	}

	return a, nil
}

// TriggerDeploy triggers a new deploy given the repository, and the branch name.
// See tests for an example input event.
func TriggerDeploy(ctx context.Context, input TriggerDeployInput) *shttp.Response {
	// Do not deploy automatically sample projects
	if input.Repo == app.SampleProjectRepo {
		return nil
	}

	// Pull requests from forks run code that the repository owner has not
	// reviewed, so they are never built automatically.
	if input.IsFork {
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
		return shttp.Error(err, fmt.Sprintf("error while fetching deploy candidates: %s", err.Error()))
	}

	candidates := []*app.DeployCandidate{}

	for _, a := range FilterDeployCandidates(input, apps) {
		if input.AppID == 0 || a.ID == input.AppID {
			candidates = append(candidates, a)
		}
	}

	if len(candidates) == 0 {
		return shttp.NoContent()
	}

	// Each deployment calls the git provider and takes a few seconds, while
	// providers give up on a webhook after ~10s and cancel the request. The
	// deployments are therefore created after the webhook is acknowledged,
	// detached from the request so a cancellation cannot drop environments.
	wd := webhookDeployer{input: input, candidates: candidates}
	bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), webhookDeployTimeout)

	runWebhookDeploys(func() {
		defer cancel()
		wd.deployAll(bg)
	})

	return shttp.OK()
}

// webhookDeployTimeout bounds the background work a single webhook triggers.
const webhookDeployTimeout = 5 * time.Minute

// runWebhookDeploys runs the deployments of a webhook. Tests replace it to
// run them synchronously.
var runWebhookDeploys = func(fn func()) { go fn() }

// webhookDeployer creates the deployments a webhook resolved to.
type webhookDeployer struct {
	input      TriggerDeployInput
	candidates []*app.DeployCandidate
}

// createGithubStatus posts a commit status. Tests replace it to avoid
// calling GitHub.
var createGithubStatus = github.CreateStatus

// deployAll deploys every candidate. A failing environment is logged and
// does not prevent the remaining ones from deploying.
func (w webhookDeployer) deployAll(ctx context.Context) {
	for _, a := range w.candidates {
		if err := w.deploy(ctx, a); err != nil {
			slog.Errorf("auto deployment failed for app id=%d, env=%s, err=%v", a.ID, a.EnvName, err)
		}
	}
}

func (w webhookDeployer) deploy(ctx context.Context, a *app.DeployCandidate) (err error) {
	// This runs outside the HTTP server, which would otherwise recover a
	// panic. An unrecovered panic in a goroutine exits the whole process.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v\n%s", r, debug.Stack())
		}
	}()

	if a.EnvDefaultBranch != w.input.Branch {
		a.ShouldPublish = false
	}

	depl := deploy.New(a.App)
	depl.PopulateFromDeployCandidate(a, deploy.DeployCandidatePayload{
		Branch:            w.input.Branch,
		CommitSha:         w.input.CommitSha,
		WebhookEvent:      w.input.payload,
		CheckoutRepo:      w.input.CheckoutRepo,
		IsFork:            w.input.IsFork,
		PullRequestNumber: w.input.PullRequestNumber,
	})

	if err := deployservice.New().Deploy(ctx, a.App, depl); err != nil {
		// The webhook has already been acknowledged, so the commit status is
		// the only place the failure shows up for the user.
		w.postStatus(a, depl, github.StatusFailure)
		return err
	}

	w.postStatus(a, depl, github.StatusPending)

	return nil
}

// postStatus posts a GitHub commit status for the deployment.
func (w webhookDeployer) postStatus(a *app.DeployCandidate, depl *deploy.Deployment, status string) {
	cnf := admin.MustConfig()

	if !a.IsGithub() || !cnf.IsGithubEnabled() {
		return
	}

	// A deployment that failed before it was inserted has no logs page.
	url := cnf.AppURL(path.Join("app", a.ID.String(), "deployments"))

	if depl.ID != 0 {
		url = cnf.DeploymentLogsURL(depl.AppID, depl.ID)
	}

	if err := createGithubStatus(a.Repo, depl.Branch, url, status); err != nil {
		slog.Errorf("error while updating github status: %s", err.Error())
	}
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
		if !buildRootChanged(input, dc) {
			continue
		}

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

// buildRootChanged reports whether a deploy candidate can be affected by the
// changed paths. Path filtering is opt-in per environment, and unknown change
// sets always preserve the existing deploy behavior.
func buildRootChanged(input TriggerDeployInput, dc *app.DeployCandidate) bool {
	if dc.BuildConfig == nil || !dc.BuildConfig.SkipUnchangedBuildRoot.ValueOrZero() {
		return true
	}

	if !input.ChangesComplete || len(input.ChangedFiles) == 0 {
		return true
	}

	matcher := newChangedPathMatcher(dc.BuildConfig)

	// An environment that builds from the repository root is affected by every
	// change, so there is nothing to filter out. Watch paths are additive and
	// must never narrow that down.
	if matcher.matchAll {
		return true
	}

	for _, file := range input.ChangedFiles {
		if matcher.matches(file) {
			return true
		}
	}

	return false
}

// changedPathMatcher decides whether a path reported by a push webhook affects
// an environment. Paths on both sides are relative to the repository root.
type changedPathMatcher struct {
	roots []string

	// matchAll is set when the build root is the repository root itself, which
	// no path can fall outside of.
	matchAll bool
}

func newChangedPathMatcher(bc *buildconf.BuildConf) *changedPathMatcher {
	m := &changedPathMatcher{matchAll: normalizeRepoPath(bc.WorkDir) == ""}

	if m.matchAll {
		return m
	}

	for _, p := range append([]string{bc.WorkDir}, bc.WatchPaths...) {
		if root := normalizeRepoPath(p); root != "" {
			m.roots = append(m.roots, root)
		}
	}

	return m
}

// matches reports whether the changed path falls under one of the watched
// roots. Repository-root files match everything because shared manifests,
// lockfiles and root configuration can affect every workspace in a monorepo.
func (m *changedPathMatcher) matches(file string) bool {
	changed := normalizeRepoPath(file)

	if changed == "" {
		return false
	}

	if !strings.Contains(changed, "/") {
		return true
	}

	for _, root := range m.roots {
		if changed == root || strings.HasPrefix(changed, root+"/") {
			return true
		}
	}

	return false
}

// normalizeRepoPath strips surrounding whitespace and slashes so that
// "apps/web", "/apps/web" and "apps/web/" compare equal. It returns an empty
// string for paths that address the repository root itself.
func normalizeRepoPath(p string) string {
	cleaned := strings.Trim(path.Clean(strings.TrimSpace(p)), "/")

	if cleaned == "." {
		return ""
	}

	return cleaned
}

// commitHasBeenBuilt checks whether there is already a build for the commit or not.
func commitHasBeenBuilt(ctx context.Context, input TriggerDeployInput) (bool, error) {
	return deploy.NewStore().IsDeploymentAlreadyBuilt(ctx, deploy.IsDeploymentAlreadyBuiltParams{
		CommitID: input.CommitSha,
		AppID:    input.AppID,
	})
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
