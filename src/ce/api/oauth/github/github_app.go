package github

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v71/github"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"golang.org/x/oauth2"
)

// Export types for easier access to packages since
// this package uses the same name with github package.

// IssueComment is a github IssueComment.
type IssueComment = github.IssueComment

// RepoStatus is a github RepoStatus.
type RepoStatus = github.RepoStatus

// RepositoryContentGetOptions is a github RepositoryContentGetOptions.
type RepositoryContentGetOptions = github.RepositoryContentGetOptions

// Package constants
const (
	StatusPending = "pending"
	StatusSuccess = "success"
	StatusFailure = "failure"
)

// Github is a wrapper around the github client to provide
// access to additional information.
type Github struct {
	*github.Client

	Owner string
	Repo  string

	// User represents the oauth2 user, that is fetched from the database.
	user *oauth.User

	Token func(context.Context) (string, error)
}

// githubAppClient returns the GitHub client, used to access the GitHub API.
func githubAppClientOld() (*github.Client, error) {
	token, err := githubAppToken()

	if err != nil {
		return nil, err
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	ctx := context.Background()
	cli := oauth2.NewClient(ctx, ts)
	return github.NewClient(cli), nil
}

// NewApp returns a new Github instance. This is used for github app actions.
func NewApp(repo string) (*Github, error) {
	if !admin.MustConfig().IsGithubEnabled() {
		slog.Info(
			"GitHub client is not configured and is trying to be accessed. " +
				"Configure it through the GITHUB_* environment variables.",
		)

		return nil, nil
	}

	owner, repo := oauth.ParseRepo(repo)
	inst, err := installationID(owner, repo)
	cnf := admin.MustConfig().AuthConfig.Github

	if err != nil {
		return nil, err
	}

	itr, err := ghinstallation.New(http.DefaultTransport, int64(cnf.AppID), int64(inst), []byte(cnf.PrivateKey))

	if err != nil {
		slog.Errorf("error while fetching github installation: %v", err)
		return nil, err
	}

	return &Github{
		Client: github.NewClient(&http.Client{Transport: itr}),
		Owner:  owner,
		Repo:   repo,
		Token:  itr.Token,
	}, nil
}

// IsPublicRepo checks whether the given repository is a public repository or not.
func IsPublicRepo(repo string) (bool, error) {
	client := github.NewClient(nil)
	owner, repoName := oauth.ParseRepo(repo)
	githubRepo, _, err := client.Repositories.Get(context.Background(), owner, repoName)

	if err != nil || githubRepo == nil {
		return false, err
	}

	return *githubRepo.Visibility == "public", nil
}

// DefaultBranch returns the default branch for the given repository.
func DefaultBranch(repo string) (string, error) {
	client, err := NewApp(repo)

	if err != nil || client == nil {
		return "", err
	}

	ghRepo, res, err := client.Repositories.Get(
		context.Background(),
		client.Owner,
		client.Repo,
	)

	if res != nil && res.StatusCode == http.StatusNotFound {
		return "", nil
	}

	if ghRepo == nil {
		return "", err
	}

	return *ghRepo.DefaultBranch, nil
}

// StormkitFile returns the contents of the stormkit.config.yml file,
// that is located at the root level of the repository.
func StormkitFile(repo, branch string) (string, error) {
	client, err := NewApp(repo)

	if err != nil || client == nil {
		return "", err
	}

	fc, _, res, err := client.Repositories.GetContents(
		context.Background(),
		client.Owner,
		client.Repo,
		"stormkit.config.yml",
		&github.RepositoryContentGetOptions{
			Ref: branch,
		},
	)

	if res != nil && res.StatusCode == http.StatusNotFound {
		return "", nil
	}

	if fc == nil || err != nil {
		return "", err
	}

	return fc.GetContent()
}

// installationID returns the installation id for the given repository, if any.
func installationID(owner, repo string) (int64, error) {
	client, err := githubAppClientOld()

	if err != nil {
		return 0, err
	}

	ctx := context.Background()
	installation, res, err := client.Apps.FindRepositoryInstallation(ctx, owner, repo)

	if err != nil || res == nil {
		errs := []string{
			"401 bad credentials",
			"404 not found",
		}

		errMsg := strings.ToLower(err.Error())

		for _, msg := range errs {
			if strings.Contains(errMsg, msg) {
				return 0, oauth.ErrRepoNotFound
			}
		}

		return 0, err
	}

	if installation != nil && installation.ID != nil {
		return *installation.ID, nil
	}

	if res.StatusCode == http.StatusNotFound {
		return 0, oauth.ErrRepoNotFound
	}

	if res.StatusCode == http.StatusForbidden {
		return 0, oauth.ErrCredsInvalidPermissions
	}

	return 0, err
}

// githubAppToken creates an application token from the private key.
func githubAppToken() (string, error) {
	cnf := admin.MustConfig().AuthConfig.Github
	privKey := cnf.PrivateKey

	token := jwt.New(jwt.GetSigningMethod("RS256"))
	token.Claims = jwt.MapClaims{
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(10 * time.Minute).Unix(),
		"iss": cnf.AppID,
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(privKey))

	if err != nil {
		return "", err
	}

	return token.SignedString(key)
}
