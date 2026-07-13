package bitbucket

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
)

// DeployKeyRequest represents a deployment key post request.
type DeployKeyRequest struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// DeployKeyResponse represents a deployment key get response.
// Read more: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-deployments/#api-repositories-workspace-repo-slug-deploy-keys-key-id-get
type DeployKeyResponse struct {
	Next   string `json:"next"`
	Values []struct {
		Key   string `json:"key"`
		Label string `json:"label"`
		Type  string `json:"type"`
	} `json:"values"`
}

// DeployKeyResponseBadRequest represents a bad request.
type DeployKeyResponseBadRequest struct {
	Key []struct {
		Message string `json:"message"`
	} `json:"key"`
}

func (b *Bitbucket) deployKeyName(a *App) string {
	return fmt.Sprintf("Stormkit Deploy Key - %d", a.ID)
}

// MustDeployKey makes sure that a deployment key exists in the repository.
func (b *Bitbucket) MustDeployKey(a *App) error {
	cnf := admin.MustConfig()

	if cnf.AuthConfig.Bitbucket.DeployKey != "" {
		return nil
	}

	if b.isDeployKeyInstalled(a) {
		return nil
	}

	return b.installDeployKey(a)
}

// isDeployKeyInstalled returns a boolean value indicating whether
// the deployment key has been already installed or not.
func (b *Bitbucket) isDeployKeyInstalled(a *App, next ...string) bool {
	owner, repo := oauth.ParseRepo(a.Repo)
	url := fmt.Sprintf("/repositories/%s/%s/deploy-keys", owner, repo)

	if len(next) > 0 {
		url = next[0]
	}

	response, err := b.get(url)

	if err != nil {
		return false
	}

	dkeys := &DeployKeyResponse{}

	if err := b.parse(response, dkeys); err != nil {
		return false
	}

	label := b.deployKeyName(a)
	key := strings.TrimSpace(a.PrivateKey.SSHPubKey())

	for _, val := range dkeys.Values {
		if val.Label == label {
			if val.Key == key {
				return true
			}
		}
	}

	if dkeys.Next != "" {
		return b.isDeployKeyInstalled(a, dkeys.Next)
	}

	return false
}

// installDeployKey installs the deploy key for the app.
func (b *Bitbucket) installDeployKey(a *App) error {
	owner, repo := oauth.ParseRepo(a.Repo)
	response, err := b.post(fmt.Sprintf("/repositories/%s/%s/deploy-keys", owner, repo), &DeployKeyRequest{
		Label: b.deployKeyName(a),
		Key:   a.PrivateKey.SSHPubKey(),
	})

	if response != nil && response.StatusCode == http.StatusBadRequest {
		payload := DeployKeyResponseBadRequest{}
		_ = b.parse(response, &payload)

		for _, msgs := range payload.Key {
			if msgs.Message == "Someone has already added that access key to this repository." {
				return nil
			}
		}
	}

	return err
}
