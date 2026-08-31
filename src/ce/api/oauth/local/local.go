// Package local provides support for apps imported from a local
// filesystem path (file:// git URLs). Unlike the GitHub/GitLab/Bitbucket
// providers there is no OAuth or API client — the repo is cloned from
// disk during deployment.
package local

import (
	"os/exec"
	"strings"
)

const ProviderName = "local"

const repoPrefix = ProviderName + "/"

// IsLocal reports whether the given App.Repo string refers to a local repo.
func IsLocal(repo string) bool {
	return strings.HasPrefix(repo, repoPrefix)
}

// Path returns the absolute filesystem path encoded in an App.Repo string.
// The stored format is "local/<path-without-leading-slash>", so a leading
// slash is prepended on the way out.
func Path(repo string) string {
	if !IsLocal(repo) {
		return ""
	}

	return "/" + strings.TrimPrefix(repo, repoPrefix)
}

// CloneURL returns the file:// URL that `git clone` can consume.
func CloneURL(repo string) string {
	if !IsLocal(repo) {
		return ""
	}

	return "file://" + Path(repo)
}

// FromURL converts a `file://<absolute-path>` URL into the stored App.Repo
// representation. Returns "" if the input is not a file:// URL.
func FromURL(fileURL string) string {
	const prefix = "file://"

	if !strings.HasPrefix(fileURL, prefix) {
		return ""
	}

	path := strings.TrimPrefix(fileURL, prefix)
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		return ""
	}

	return repoPrefix + path
}

// DefaultBranch returns the local repo's current branch via `git symbolic-ref`.
// Falls back to "main" when the path is not a valid git repo.
func DefaultBranch(repo string) (string, error) {
	out, err := exec.Command("git", "-C", Path(repo), "symbolic-ref", "--short", "HEAD").Output()

	if err != nil {
		return "main", err
	}

	return strings.TrimSpace(string(out)), nil
}
