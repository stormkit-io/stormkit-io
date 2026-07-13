package gitlab

import (
	"fmt"

	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/xanzy/go-gitlab"
)

// DefaultBranch returns the default branch of a given repository.
func (g *Gitlab) DefaultBranch(repo string) (string, error) {
	owner, project := oauth.ParseRepo(repo)
	pid := fmt.Sprintf("%s/%s", owner, project)
	repository, _, err := g.Projects.GetProject(pid, &gitlab.GetProjectOptions{})

	if repository == nil || err != nil {
		return "", err
	}

	return repository.DefaultBranch, nil
}

// Files returns the list of files in a gitlab repository.
func (g *Gitlab) ReadFile(repo, branch, fileName string) ([]byte, error) {
	owner, project := oauth.ParseRepo(repo)
	pid := fmt.Sprintf("%s/%s", owner, project)
	options := &gitlab.GetRawFileOptions{Ref: &branch}
	content, _, err := g.RepositoryFiles.GetRawFile(pid, fileName, options)

	if err != nil {
		return nil, err
	}

	return content, err
}
