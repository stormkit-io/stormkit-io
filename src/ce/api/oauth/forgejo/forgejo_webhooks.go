package forgejo

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const hooksPath = "/app/webhooks/forgejo"

// InstallWebhooksParams represents the parameters for InstallWebhooks.
type InstallWebhooksParams struct {
	// Repo is the Stormkit formatted repository address.
	Repo string

	// AppID is the Stormkit app the webhook deploys.
	AppID types.ID
}

type hook struct {
	ID     int64             `json:"id"`
	Config map[string]string `json:"config"`
}

// InstallWebhooks installs a webhook that triggers deployments of the given
// app on push and pull request events. Each app gets its own hook,
// identified by the app secret in its URL. It reports whether a hook
// was created.
func (f *Forgejo) InstallWebhooks(p InstallWebhooksParams) (bool, error) {
	secret := utils.EncryptID(p.AppID)

	if secret == "" {
		return false, errors.New("cannot generate the webhook secret")
	}

	hooks := []hook{}
	path := fmt.Sprintf("/repos/%s/hooks", f.repoPath(p.Repo))

	if _, err := f.do(request{method: http.MethodGet, path: path, out: &hooks}); err != nil {
		return false, err
	}

	manager := hookManager{baseURL: admin.MustConfig().WebhooksURL(hooksPath), appID: p.AppID}

	if manager.isInstalled(hooks) {
		return false, nil
	}

	// The "gitea" hook type is understood by both Forgejo and Gitea.
	_, err := f.do(request{
		method: http.MethodPost,
		path:   path,
		body: map[string]any{
			"type":   "gitea",
			"active": true,
			"events": []string{"push", "pull_request"},
			"config": map[string]string{
				"url":          manager.baseURL + "/" + secret,
				"content_type": "json",
			},
		},
	})

	return err == nil, err
}

// hookManager inspects the hooks registered on a repository.
type hookManager struct {
	baseURL string
	appID   types.ID
}

// isInstalled reports whether a hook carrying this app's secret exists.
func (m hookManager) isInstalled(hooks []hook) bool {
	for _, h := range hooks {
		secret, found := strings.CutPrefix(h.Config["url"], m.baseURL+"/")

		if !found {
			continue
		}

		if id, err := utils.DecryptID(secret); err == nil && id == m.appID {
			return true
		}
	}

	return false
}
