package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
)

type RepoInterface interface {
	Checkout(context.Context) error
	CommitInfo() map[string]string
	IsGithub() bool
	IsGitlab() bool
	IsBitbucket() bool
	Address() string
}

type Repo struct {
	dir         string // The directory where the repo is checked out
	keysDir     string
	address     string
	accessToken string
	provider    string
	branch      string // The branch to checkout
	workDir     string
	vars        map[string]string
	reporter    *ReporterModel
}

const REPO_TYPE_GITHUB = "github"
const REPO_TYPE_GITLAB = "gitlab"
const REPO_TYPE_BITBUCKET = "bitbucket"

var DefaultRepo RepoInterface

// NewRepo creates a new repo instance from the given address and access token.
func NewRepo(opts RunnerOpts) RepoInterface {
	if DefaultRepo != nil {
		return DefaultRepo
	}

	repo := Repo{
		dir:         opts.Repo.Dir,
		keysDir:     opts.KeysDir,
		address:     opts.Repo.Address,
		accessToken: opts.Repo.AccessToken,
		branch:      opts.Repo.Branch,
		workDir:     opts.WorkDir,
		vars:        opts.Build.EnvVars,
		reporter:    opts.Reporter,
	}

	if strings.HasPrefix(repo.address, "https://github.com") {
		repo.provider = REPO_TYPE_GITHUB
	} else if strings.HasPrefix(repo.address, "https://gitlab.com") {
		repo.provider = REPO_TYPE_GITLAB
	} else if strings.HasPrefix(repo.address, "git@bitbucket.org") {
		repo.provider = REPO_TYPE_BITBUCKET
	}

	return repo
}

// Address returns the repository address.
func (r Repo) Address() string {
	return r.address
}

// Checkout checks out the repository
func (r Repo) Checkout(ctx context.Context) error {
	var addr string
	var ssh string

	addr = r.address

	if r.accessToken != "" {
		if r.IsGithub() {
			addr = strings.Replace(r.address, "https://", fmt.Sprintf("https://x-access-token:%s@", r.accessToken), 1)
		} else if r.IsGitlab() {
			addr = strings.Replace(r.address, "https://", fmt.Sprintf("https://oauth2:%s@", r.accessToken), 1)
		} else if r.IsBitbucket() {
			err := r.createSSHKeys()

			// In case the createSSHKeys return an error, it's likely that the user has provided
			// a custom access token.
			if err != nil {
				addr = strings.Replace(r.address, "git@bitbucket.org:", fmt.Sprintf("https://x-token-auth:%s@bitbucket.org/", r.accessToken), 1)
			} else {
				ssh = fmt.Sprintf(`GIT_SSH_COMMAND="ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i %s/id_rsa"`, r.keysDir)
			}
		}
	}

	r.reporter.AddStep(fmt.Sprintf("checkout %s", r.branch))

	// Add this system variable
	r.vars["SK_BRANCH_NAME"] = r.branch

	// See https://github.com/golang/go/issues/38268#issuecomment-609562062 for the progress flag
	cmd := sys.Command(
		ctx,
		sys.CommandOpts{
			Name: "git",
			Args: []string{
				"clone", addr,
				"--depth", "1",
				"--progress", "--single-branch",
				"--branch", r.branch, r.dir,
			},
			Env:    PrepareEnvVars(r.vars),
			Stdout: r.reporter.File(),
			Stderr: r.reporter.File(),
		},
	)

	if ssh != "" {
		cmd = sys.Command(ctx, sys.CommandOpts{
			Name:   "sh",
			Args:   []string{"-c", fmt.Sprintf("%s %s", ssh, cmd.String())},
			Env:    PrepareEnvVars(r.vars),
			Stdout: r.reporter.File(),
			Stderr: r.reporter.File(),
		})
	}

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

// HeadSHA returns information on the latest commit's SHA.
func (r Repo) HeadSHA() string {
	cmd := sys.Command(context.Background(), sys.CommandOpts{
		Name: "git",
		Args: []string{"rev-parse", "HEAD"},
		Dir:  r.dir,
		Env:  PrepareEnvVars(r.vars),
	})

	msg, _ := cmd.Output()
	return strings.ReplaceAll(string(msg), "\n", "")
}

// AuthorInfo returns information on the latest commit's author.
func (r Repo) AuthorInfo() string {
	cmd := sys.Command(context.Background(), sys.CommandOpts{
		Name: "git",
		Args: []string{"--no-pager", "show", "-s", "--format='%an <%ae>'", "HEAD"},
		Dir:  r.dir,
		Env:  PrepareEnvVars(r.vars),
	})

	msg, _ := cmd.Output()
	return strings.Trim(strings.ReplaceAll(string(msg), "\n", ""), "'")
}

// CommitMsg returns the HEAD commit message.
func (r Repo) CommitMsg() string {
	cmd := sys.Command(context.Background(), sys.CommandOpts{
		Name: "git",
		Args: []string{"log", "-1", "--pretty=%B"},
		Dir:  r.dir,
		Env:  PrepareEnvVars(r.vars),
	})

	msg, _ := cmd.Output()
	return strings.Split(string(msg), "\n\n")[0]
}

// CommitInfo returns a map of information related to the latest commit.
func (r Repo) CommitInfo() map[string]string {
	info := map[string]string{
		"sha":     r.HeadSHA(),
		"author":  r.AuthorInfo(),
		"message": r.CommitMsg(),
	}

	r.vars["SK_COMMIT_SHA"] = info["sha"]
	return info
}

// IsGithub returns true if the provider is Github.
func (r Repo) IsGithub() bool {
	return r.provider == REPO_TYPE_GITHUB
}

// IsGitlab returns true if the provider is Gitlab.
func (r Repo) IsGitlab() bool {
	return r.provider == REPO_TYPE_GITLAB
}

// IsBitbucket returns true if the provider is Bitbucket.
func (r Repo) IsBitbucket() bool {
	return r.provider == REPO_TYPE_BITBUCKET
}

func (r Repo) createSSHKeys() error {
	creds, err := utils.DecodeString(r.accessToken)

	if err != nil {
		return err
	}

	pieces := strings.Split(string(creds), "|")

	if len(pieces) < 2 {
		return errors.New("invalid access token given")
	}

	_, publicKey, privateKey := pieces[0], pieces[1], pieces[2]

	err = os.WriteFile(path.Join(r.keysDir, "id_rsa.pub"), []byte(publicKey), 0644)

	if err != nil {
		return err
	}

	err = os.WriteFile(path.Join(r.keysDir, "id_rsa"), []byte(privateKey), 0600)

	if err != nil {
		return err
	}

	return nil
}
