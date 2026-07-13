package bitbucket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
)

// FilesEntry represents a tree entry with type commit_file.
type FilesEntry struct {
	Type string
	Path string
}

// FilesResponse represents a file response.
type FilesResponse struct {
	Next   string
	Values []FilesEntry
}

// BranchShaTarget represents the hash code of the branch.
type BranchShaTarget struct {
	Hash string `json:"hash"`
}

// BranchShaResponse represents a branch info response.
type BranchShaResponse struct {
	Target BranchShaTarget `json:"target"`
}

// StormkitFile reads the stormkit.config.yml file at the root level of the
// repository and returns its content, if any.
func (b *Bitbucket) StormkitFile(a *App, branch string) (string, error) {
	owner, repo := oauth.ParseRepo(a.Repo)
	response, err := b.get(fmt.Sprintf("/repositories/%s/%s/src/%s/stormkit.config.yml", owner, repo, branch))

	if err != nil {
		return "", err
	}

	by, err := io.ReadAll(response.Body)
	return string(by), err
}

// ReadFile reads a file content in a bitbucket repository.
func (b *Bitbucket) ReadFile(a *App, branch, fileNameIncludingPath string) ([]byte, error) {
	return b.readFile(a, branch, fileNameIncludingPath)
}

// branchSha returns the sha of the head commit of the branch. We need this method
// because Bitbucket does not support slashes in names.
func (b *Bitbucket) branchSha(a *App, branch string) (string, error) {
	owner, repo := oauth.ParseRepo(a.Repo)
	res, err := b.get(fmt.Sprintf("/repositories/%s/%s/refs/branches/%s", owner, repo, branch))
	data := &BranchShaResponse{}

	if err != nil || res == nil {
		return "", err
	}

	err = json.NewDecoder(res.Body).Decode(data)
	return data.Target.Hash, err
}

// readFile reads a file content and returns it as a string.
func (b *Bitbucket) readFile(a *App, branch, fileNameIncludingPath string) ([]byte, error) {
	owner, repo := oauth.ParseRepo(a.Repo)
	sha, err := b.branchSha(a, branch)

	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("/repositories/%s/%s/src/%s/%s", owner, repo, sha, fileNameIncludingPath)

	response, err := b.get(uri)

	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			return nil, nil
		}

		return nil, err
	}

	defer response.Body.Close()
	bytes, err := io.ReadAll(response.Body)

	if err != nil {
		return nil, err
	}

	return bytes, nil
}
