package gitlab

import (
	gitlab "github.com/xanzy/go-gitlab"
)

// RepoSize returns the size of the repository in bytes, as reported by GitLab.
// Project statistics are only returned to members with at least Reporter
// access, so this yields 0 when the token cannot see them.
func (g *Gitlab) RepoSize(repo string) (int64, error) {
	project, _, err := g.Projects.GetProject(g.SanitizeRepo(repo), &gitlab.GetProjectOptions{
		Statistics: gitlab.Ptr(true),
	})

	if err != nil {
		return 0, err
	}

	if project == nil || project.Statistics == nil {
		return 0, nil
	}

	return project.Statistics.RepositorySize, nil
}
