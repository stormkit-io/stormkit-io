package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
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

	limit := maxRepoSize()

	if err := r.clone(ctx, cloneOpts{addr: addr, ssh: ssh, limit: limit}); err != nil {
		return err
	}

	// The clone above left the working tree empty, so the repository can be
	// measured exactly before a single file of it is written to disk.
	if limit > 0 {
		size, err := repoSizer{dir: r.dir, vars: r.vars}.total(ctx)

		if err != nil {
			return err
		}

		if size > limit {
			slog.Errorf("rejecting a %s repository, over the %s cap", humanBytes(size), humanBytes(limit))
			return ErrRepoTooLarge{size: size, limit: limit}
		}
	}

	return r.populateWorkTree(ctx)
}

type cloneOpts struct {
	addr  string
	ssh   string
	limit int64
}

// clone downloads the repository without populating the working tree. When a
// cap applies, the download runs under prlimit so the kernel stops the pack at
// the limit; that bound is inherited by every process git spawns, including a
// git index-pack that outlives its parent.
func (r Repo) clone(ctx context.Context, opts cloneOpts) error {
	args := []string{
		"clone", opts.addr,
		"--depth", "1",
		"--progress", "--single-branch", "--no-checkout",
		"--branch", r.branch, r.dir,
	}

	name := "git"

	// prlimit is util-linux, so it is absent on macOS and could go missing
	// from a future image. Without it the download is unbounded until the
	// measurement below runs, which is a weaker guarantee but not a broken
	// deploy, so fall back rather than fail.
	if opts.limit > 0 {
		if prlimit, err := exec.LookPath("prlimit"); err == nil {
			name = prlimit
			args = append([]string{
				fmt.Sprintf("--fsize=%d", opts.limit),
				"--core=0",
				"git",
			}, args...)
		} else {
			slog.Errorf("prlimit not found, the download is measured but not bounded: %v", err)
		}
	}

	stderr := &cloneStderr{}

	// Only tee when a cap applies: an uncapped clone has nothing to attribute,
	// and this keeps its command identical to what it has always been.
	var stderrW io.Writer = r.reporter.File()

	if opts.limit > 0 {
		stderrW = io.MultiWriter(r.reporter.File(), stderr)
	}

	// See https://github.com/golang/go/issues/38268#issuecomment-609562062 for the progress flag
	cmd := sys.Command(ctx, sys.CommandOpts{
		Name:   name,
		Args:   args,
		Env:    PrepareEnvVars(r.vars),
		Stdout: r.reporter.File(),
		Stderr: stderrW,
	})

	if opts.ssh != "" {
		cmd = sys.Command(ctx, sys.CommandOpts{
			Name:   "sh",
			Args:   []string{"-c", fmt.Sprintf("%s %s", opts.ssh, cmd.String())},
			Env:    PrepareEnvVars(r.vars),
			Stdout: r.reporter.File(),
			Stderr: stderrW,
		})
	}

	if err := cmd.Run(); err != nil {
		if opts.limit > 0 && stderr.sawFileSizeLimit() {
			return ErrRepoTooLarge{limit: opts.limit}
		}

		return err
	}

	return nil
}

// populateWorkTree writes out the files the clone deliberately skipped. The
// index is empty after --no-checkout, so this has to name HEAD explicitly.
func (r Repo) populateWorkTree(ctx context.Context) error {
	cmd := sys.Command(ctx, sys.CommandOpts{
		Name:   "git",
		Args:   []string{"checkout", "-f", "HEAD"},
		Dir:    r.dir,
		Env:    PrepareEnvVars(r.vars),
		Stdout: r.reporter.File(),
		Stderr: r.reporter.File(),
	})

	return cmd.Run()
}

// cloneStderr keeps the tail of a clone's stderr so a failure can be
// attributed after the fact. Only the tail is retained; a clone can emit
// megabytes of progress output.
type cloneStderr struct {
	mu   sync.Mutex
	tail []byte
}

const cloneStderrTail = 4 << 10

func (c *cloneStderr) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tail = append(c.tail, p...)

	if len(c.tail) > cloneStderrTail {
		c.tail = c.tail[len(c.tail)-cloneStderrTail:]
	}

	return len(p), nil
}

// sawFileSizeLimit reports whether the clone failed the way it does when
// RLIMIT_FSIZE stops the pack.
//
// The kernel kills git index-pack, not the git clone we waited on: the parent
// sees its child fail and exits 128 normally, so there is no SIGXFSZ in the
// wait status to key off. What is left is git's own message. This decides the
// wording of an error, never whether the cap is enforced -- prlimit has
// already stopped the write by the time we get here -- so a miss costs a
// clearer message, not a filled disk.
func (c *cloneStderr) sawFileSizeLimit() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, sig := range []string{
		"invalid index-pack output",
		"File too large",
		"File size limit exceeded",
	} {
		if bytes.Contains(c.tail, []byte(sig)) {
			return true
		}
	}

	return false
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
