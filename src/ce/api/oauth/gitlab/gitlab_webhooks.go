package gitlab

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	gl "github.com/xanzy/go-gitlab"
)

var hooksPath = "/app/webhooks/gitlab"

// InstallWebhooks installs the webhooks for the given repository. The webhooks
// will be used to trigger deployments on push and merge request events.
// `repo` is the Stormkit formatted repository address.
func (g *Gitlab) InstallWebhooks(repo string) (bool, error) {
	owner, project := oauth.ParseRepo(repo)
	repository := fmt.Sprintf("%s/%s", owner, project)

	if has, _ := g.hooksInstalled(repository); has {
		return false, nil
	}

	hook, res, err := g.Projects.AddProjectHook(repository, &gl.AddProjectHookOptions{
		URL:                 utils.Ptr(admin.MustConfig().WebhooksURL(hooksPath)),
		PushEvents:          utils.Ptr(true),
		MergeRequestsEvents: utils.Ptr(true),
		NoteEvents:          utils.Ptr(true),
	})

	if res != nil && res.StatusCode == http.StatusUnauthorized {
		return false, oauth.ErrNotAuthorized
	}

	return hook != nil, err
}

// hooksInstalled checks whether the hooks are already installed or not.
func (g *Gitlab) hooksInstalled(repo string) (bool, error) {
	hooks, _, err := g.Projects.ListProjectHooks(repo, &gl.ListProjectHooksOptions{
		PerPage: 100,
	})

	if err != nil {
		return false, err
	}

	cnf := admin.MustConfig()

	if len(hooks) > 0 {
		for _, hook := range hooks {
			if strings.Contains(hook.URL, cnf.WebhooksURL(hooksPath)) {
				return true, nil
			}
		}
	}

	return false, nil
}
