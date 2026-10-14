package forgejo

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// InstallDeployKeyParams represents the parameters for InstallDeployKey.
type InstallDeployKeyParams struct {
	// Repo is the Stormkit formatted repository address.
	Repo string

	// AppID is the Stormkit app the key belongs to.
	AppID types.ID

	// PublicKey is the app's public key in authorized_keys format.
	PublicKey string
}

type deployKey struct {
	Key string `json:"key"`
}

// InstallDeployKey adds the app's public key to the repository as a read-only
// deploy key, unless it is already there. Builds clone with this key, so the
// user's token, which Forgejo cannot scope, never reaches them.
func (f *Forgejo) InstallDeployKey(p InstallDeployKeyParams) error {
	path := fmt.Sprintf("/repos/%s/keys", f.repoPath(p.Repo))
	publicKey := strings.TrimSpace(p.PublicKey)
	keys := []deployKey{}

	if _, err := f.do(request{method: http.MethodGet, path: path, out: &keys}); err != nil {
		return err
	}

	for _, key := range keys {
		// Forgejo appends a comment to the stored key.
		if strings.HasPrefix(strings.TrimSpace(key.Key), publicKey) {
			return nil
		}
	}

	_, err := f.do(request{
		method: http.MethodPost,
		path:   path,
		body: map[string]any{
			"title":     fmt.Sprintf("Stormkit Deploy Key - %d", p.AppID),
			"key":       publicKey,
			"read_only": true,
		},
	})

	return err
}
