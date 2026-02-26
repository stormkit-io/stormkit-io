package apphandlers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apphandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
)

type InboundWebhooksSuite struct {
	suite.Suite
	*factory.Factory

	// conn databasetest.TestDB
	list []*app.DeployCandidate
}

func (s *InboundWebhooksSuite) BeforeTest(suiteName, _ string) {
	myApp := &app.MyApp{
		App: &app.App{},
	}

	s.list = []*app.DeployCandidate{
		{
			MyApp:            myApp,
			EnvName:          "development",
			EnvDefaultBranch: "staging",
		},
		{
			MyApp:            myApp,
			EnvName:          "production",
			EnvDefaultBranch: "master",
		},
		{
			MyApp:            myApp,
			EnvName:          "testing",
			EnvDefaultBranch: "testing",
		},
		{
			MyApp:              myApp,
			EnvName:            "auto-deploy-branch",
			EnvDefaultBranch:   "auto-deploys",
			AutoDeployBranches: null.NewString("^(?!dependabot).+", true),
		},
	}
}

func (s *InboundWebhooksSuite) Test_TriggerDeploy() {
	commitInput := apphandlers.TriggerDeployInput{
		Repo:         "github/stormkit-io/sample-project",
		CheckoutRepo: "github/stormkit-io/sample-project",
		Branch:       "main",
		Message:      "chore: use suggested method",
		EventType:    "commit",
	}

	pullRequestInput := apphandlers.TriggerDeployInput{
		Repo:              "github/stormkit-io/sample-project",
		CheckoutRepo:      "github/stormkit-io/sample-project",
		IsFork:            false,
		Branch:            "example-pr",
		Message:           "chore: example pr",
		EventType:         "pull_request",
		PullRequestNumber: 2,
	}

	// TODO: Create real tests from these inputs
	s.NotEmpty(pullRequestInput)
	s.NotEmpty(commitInput)
}

func (s *InboundWebhooksSuite) Test_MatchPattern() {
	a := assert.New(s.T())
	a.True(apphandlers.MatchPattern("^dependabot/.*", "dependabot/bump-node-forge-1.4.5"))
	a.True(apphandlers.MatchPattern("^(dependabot|renovate)/.*", "dependabot/bump-node-forge-1.4.5"))
	a.True(apphandlers.MatchPattern("^(dependabot|renovate)/.*", "renovate/bump-node-forge-1.4.5"))
	a.False(apphandlers.MatchPattern("^(?!dependabot|renovate).*/.*", "renovate/bump-node-forge-1.4.5"))
	a.False(apphandlers.MatchPattern("^(?!dependabot|renovate).*/.*", "dependabot/bump-node-forge-1.4.5"))
	a.False(apphandlers.MatchPattern("^(?!dependabot|renovate).*/.*", "release"))
	a.True(apphandlers.MatchPattern("^(?!dependabot|renovate).*/.*", "release/staging"))
	a.True(apphandlers.MatchPattern("^(?!dependabot).+", "auto-deploy-branches"))
	a.True(apphandlers.MatchPattern("my-branch", "hello-my-branch"))
	a.False(apphandlers.MatchPattern("^my-branch", "hello-my-branch"))
	a.False(apphandlers.MatchPattern("my-branch", "my-b"))
	a.True(apphandlers.MatchPattern("release-*", "release-staging"))
	a.True(apphandlers.MatchPattern(`^chore\(release\):.+`, "chore(release): version 10.504.21"))
	a.False(apphandlers.MatchPattern(`^chore(release):.+`, "chore(release): version 10.504.21"))
	a.False(apphandlers.MatchPattern(`\[deploy\]`, "chore: remove env variable"))
	a.True(apphandlers.MatchPattern(`\[deploy\]`, "chore: remove env variable [deploy]"))
}

func (s *InboundWebhooksSuite) Test_FilterDeployCandidates_AutoDeploy() {
	s.list[0].AutoDeployBranches = null.NewString("some-regex", true)
	s.list[1].AutoDeployBranches = null.NewString("some-other-regex", true)
	s.list[2].AutoDeployBranches = null.NewString("another-regex", true)
	s.list[3].AutoDeployBranches = null.NewString("", false)

	c := apphandlers.FilterDeployCandidates(apphandlers.TriggerDeployInput{
		Branch: "staging",
	}, s.list)

	s.Len(c, 2)
	s.Equal("development", c[0].EnvName)
	s.Equal("auto-deploy-branch", c[1].EnvName)

	c = apphandlers.FilterDeployCandidates(apphandlers.TriggerDeployInput{
		Branch: "master",
	}, s.list)

	s.Len(c, 2)
	s.Equal("production", c[0].EnvName)
	s.Equal("master", c[0].EnvDefaultBranch)
	s.Equal("auto-deploy-branch", c[1].EnvName)
	s.Equal("", c[1].AutoDeployBranches.ValueOrZero())

	c = apphandlers.FilterDeployCandidates(apphandlers.TriggerDeployInput{
		Branch:            "feature-branch",
		PullRequestNumber: 201,
	}, s.list)

	s.Equal("auto-deploy-branch", c[0].EnvName)
	s.Len(c, 1)
}

func (s *InboundWebhooksSuite) Test_FilterDeployCandidates_AutoDeployBranchesConfig() {
	myApp := &app.MyApp{
		App: &app.App{},
	}

	list := []*app.DeployCandidate{
		{
			MyApp:              myApp,
			EnvName:            "development",
			EnvDefaultBranch:   "staging",
			AutoDeployBranches: null.NewString("branch-1-*", true),
		},
		{
			MyApp:              myApp,
			EnvName:            "production",
			EnvDefaultBranch:   "master",
			AutoDeployBranches: null.NewString("branch-2-*", true),
		},
		{
			MyApp:              myApp,
			EnvName:            "testing",
			EnvDefaultBranch:   "testing",
			AutoDeployBranches: null.NewString("branch-*", true),
		},
	}

	c := apphandlers.FilterDeployCandidates(apphandlers.TriggerDeployInput{
		Branch: "staging",
	}, list)

	s.Equal("development", c[0].EnvName)
	s.Len(c, 1)

	c = apphandlers.FilterDeployCandidates(apphandlers.TriggerDeployInput{
		Branch:            "branch-1-a",
		PullRequestNumber: 201,
	}, list)

	s.Len(c, 2)
	s.Equal("development", c[0].EnvName)
	s.Equal("testing", c[1].EnvName)

	c = apphandlers.FilterDeployCandidates(apphandlers.TriggerDeployInput{
		Branch:            "branch-3",
		PullRequestNumber: 201,
	}, list)

	s.Equal("testing", c[0].EnvName)
	s.Len(c, 1)
}

func (s *InboundWebhooksSuite) Test_FilterDeployCandidates_AutoDeployCommitsConfig() {
	myApp := &app.MyApp{
		App: &app.App{},
	}

	list := []*app.DeployCandidate{
		{
			MyApp:             myApp,
			EnvName:           "development",
			EnvDefaultBranch:  "staging",
			AutoDeployCommits: null.NewString("release:*", true),
		},
		{
			MyApp:             myApp,
			EnvName:           "development-2",
			EnvDefaultBranch:  "staging",
			AutoDeployCommits: null.NewString("something-else:*", true),
		},
	}

	// The commit should only be checked if the branches match
	c := apphandlers.FilterDeployCandidates(apphandlers.TriggerDeployInput{
		Message: "release: hello world",
		Branch:  "staging",
	}, list)

	s.Equal("development", c[0].EnvName)
	s.Len(c, 1)

	// The commit should not be checked if branches do not match
	c = apphandlers.FilterDeployCandidates(apphandlers.TriggerDeployInput{
		Message: "release: hello world",
		Branch:  "some-random-branch",
	}, list)

	s.Len(c, 0)
}

func TestIncomingWebhooks(t *testing.T) {
	suite.Run(t, &InboundWebhooksSuite{})
}
