package gitlab

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	gl "github.com/xanzy/go-gitlab"
)

var hooksPath = "/app/webhooks/gitlab"

// InstallWebhooksParams represents the parameters for InstallWebhooks.
type InstallWebhooksParams struct {
	// Repo is the Stormkit formatted repository address.
	Repo string

	// AppID is the Stormkit app the webhook deploys.
	AppID types.ID
}

// InstallWebhooks installs the webhooks for the given app. The webhooks
// will be used to trigger deployments on push and merge request events.
// Each app gets its own hook, identified by the app secret in its URL.
func (g *Gitlab) InstallWebhooks(p InstallWebhooksParams) (bool, error) {
	secret := utils.EncryptID(p.AppID)

	if secret == "" {
		return false, errors.New("cannot generate the webhook secret")
	}

	owner, project := oauth.ParseRepo(p.Repo)
	repository := fmt.Sprintf("%s/%s", owner, project)
	hookURL := admin.MustConfig().WebhooksURL(fmt.Sprintf("%s/%s", hooksPath, secret))

	// A failed listing is not fatal: the hook is added below, as before.
	hooks, res, _ := g.Projects.ListProjectHooks(repository, &gl.ListProjectHooksOptions{
		PerPage: 100,
	})

	if res != nil && res.StatusCode == http.StatusUnauthorized {
		return false, oauth.ErrNotAuthorized
	}

	manager := hookManager{baseURL: admin.MustConfig().WebhooksURL(hooksPath), appID: p.AppID}

	if manager.isInstalled(hooks) {
		return false, nil
	}

	// Hooks registered before webhooks were verified carry no secret and are
	// now rejected. Point the first one at this app instead of adding another.
	if legacy := manager.legacyHook(hooks); legacy != nil {
		_, _, err := g.Projects.EditProjectHook(repository, legacy.ID, &gl.EditProjectHookOptions{
			URL:                 utils.Ptr(hookURL),
			PushEvents:          utils.Ptr(true),
			MergeRequestsEvents: utils.Ptr(true),
			NoteEvents:          utils.Ptr(true),
		})

		return err == nil, err
	}

	hook, res, err := g.Projects.AddProjectHook(repository, &gl.AddProjectHookOptions{
		URL:                 utils.Ptr(hookURL),
		PushEvents:          utils.Ptr(true),
		MergeRequestsEvents: utils.Ptr(true),
		NoteEvents:          utils.Ptr(true),
	})

	if res != nil && res.StatusCode == http.StatusUnauthorized {
		return false, oauth.ErrNotAuthorized
	}

	return hook != nil, err
}

// hookManager inspects the hooks registered on a GitLab project.
type hookManager struct {
	baseURL string
	appID   types.ID
}

// isInstalled reports whether a hook carrying this app's secret exists.
func (m hookManager) isInstalled(hooks []*gl.ProjectHook) bool {
	for _, hook := range hooks {
		secret, found := strings.CutPrefix(hook.URL, m.baseURL+"/")

		if !found {
			continue
		}

		if id, err := utils.DecryptID(secret); err == nil && id == m.appID {
			return true
		}
	}

	return false
}

// legacyHook returns a Stormkit hook that has no app secret in its URL.
func (m hookManager) legacyHook(hooks []*gl.ProjectHook) *gl.ProjectHook {
	for _, hook := range hooks {
		if strings.TrimSuffix(hook.URL, "/") == m.baseURL {
			return hook
		}
	}

	return nil
}
