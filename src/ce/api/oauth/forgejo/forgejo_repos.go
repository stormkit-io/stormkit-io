package forgejo

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Repository is a repository returned by the API.
type Repository struct {
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`

	// SSHURL is empty when the instance has SSH disabled.
	SSHURL string `json:"ssh_url"`

	// Size is the repository size in KiB.
	Size int64 `json:"size"`
}

// ListReposParams represents the parameters for ListRepos.
type ListReposParams struct {
	Page    int
	PerPage int
}

// ListRepos returns a page of repositories the user has access to and
// whether there is a next page.
func (f *Forgejo) ListRepos(p ListReposParams) ([]Repository, bool, error) {
	repos := []Repository{}

	res, err := f.do(request{
		method: http.MethodGet,
		path:   "/user/repos",
		query: url.Values{
			"page":  {strconv.Itoa(p.Page)},
			"limit": {strconv.Itoa(p.PerPage)},
		},
		out: &repos,
	})

	if err != nil {
		return nil, false, err
	}

	// The instance may cap the page size below the requested one, so the
	// next page is taken from the pagination links rather than computed.
	if link := res.Header.Get("Link"); link != "" {
		return repos, strings.Contains(link, `rel="next"`), nil
	}

	return repos, len(repos) == p.PerPage, nil
}

// Repository returns the given Stormkit formatted repository.
func (f *Forgejo) Repository(repo string) (*Repository, error) {
	repository := &Repository{}

	if _, err := f.do(request{method: http.MethodGet, path: "/repos/" + f.repoPath(repo), out: repository}); err != nil {
		return nil, err
	}

	return repository, nil
}

// DefaultBranch returns the default branch of a given repository.
func (f *Forgejo) DefaultBranch(repo string) (string, error) {
	repository, err := f.Repository(repo)

	if err != nil {
		return "", err
	}

	return repository.DefaultBranch, nil
}

// RepoSize returns the size of the repository in bytes.
func (f *Forgejo) RepoSize(repo string) (int64, error) {
	repository, err := f.Repository(repo)

	if err != nil {
		return 0, err
	}

	return repository.Size << 10, nil
}

// CreatePullRequestComment leaves a comment on the given pull request.
// Pull requests share their number space with issues.
func (f *Forgejo) CreatePullRequestComment(p CreatePullRequestCommentParams) error {
	_, err := f.do(request{
		method: http.MethodPost,
		path:   fmt.Sprintf("/repos/%s/issues/%d/comments", f.repoPath(p.Repo), p.Number),
		body:   map[string]string{"body": p.Body},
	})

	return err
}

// CreatePullRequestCommentParams represents the parameters for CreatePullRequestComment.
type CreatePullRequestCommentParams struct {
	Repo   string
	Number int64
	Body   string
}
