package bitbucket

import (
	"fmt"
	"strings"
)

// RepoSize returns the size of the repository in bytes, as reported by
// Bitbucket.
func (b *Bitbucket) RepoSize(repo string) (int64, error) {
	res, err := b.get(fmt.Sprintf("/repositories/%s", strings.Replace(repo, "bitbucket/", "", 1)))

	if err != nil {
		return 0, err
	}

	defer res.Body.Close()

	var payload struct {
		Size int64 `json:"size"`
	}

	if err := b.parse(res, &payload); err != nil {
		return 0, err
	}

	return payload.Size, nil
}
