package bitbucket

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
)

// PullRequestCommentBody represents the body to be sent.
type PullRequestCommentBody struct {
	Raw string `json:"raw"`
}

// PullRequestCommentRequest represents a pull request comment payload
// that is going to be sent to the bitbucket API.
type PullRequestCommentRequest struct {
	Body PullRequestCommentBody `json:"content"`
}

// PullRequestCommentResponse represents a pull request comment
// response from bitbucket API.
type PullRequestCommentResponse struct {
	ID int64 `json:"id"`
}

// PullRequestComment creates a new comment on the pull request.
func (b *Bitbucket) PullRequestComment(a *App, body string, prNumber int64) (*PullRequestCommentResponse, error) {
	owner, repo := oauth.ParseRepo(a.Repo)

	res, err := b.post(fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments", owner, repo, prNumber), PullRequestCommentRequest{
		PullRequestCommentBody{
			Raw: body,
		},
	})

	if err != nil {
		return nil, err
	}

	comment := &PullRequestCommentResponse{}

	if err := json.NewDecoder(res.Body).Decode(comment); err != nil {
		return nil, err
	}

	return comment, nil
}

// PullRequestRemoveComment creates a new comment on the pull request.
func (b *Bitbucket) PullRequestRemoveComment(a *App, prNumber, commentID int64) error {
	owner, repo := oauth.ParseRepo(a.Repo)

	res, err := b.delete(fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments/%d", owner, repo, prNumber, commentID))

	if res != nil {
		b1, _ := io.ReadAll(res.Body)
		fmt.Println(string(b1))
	}

	return err
}
