package bitbucket

import (
	"encoding/json"
	"fmt"

	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
)

type defaultBranchResponse struct {
	MainBranch struct {
		Type string `json:"type"`
		Name string `json:"name"`
	} `json:"mainbranch"`
}

// DefaultBranch returns the default branch for the given repository.
func (b *Bitbucket) DefaultBranch(repo string) (string, error) {
	owner, repo := oauth.ParseRepo(repo)
	response, err := b.get(fmt.Sprintf("/repositories/%s/%s", owner, repo))

	if err != nil {
		return "", err
	}

	dbr := &defaultBranchResponse{}

	if err := json.NewDecoder(response.Body).Decode(dbr); err != nil {
		return "", err
	}

	return string(dbr.MainBranch.Name), err
}
