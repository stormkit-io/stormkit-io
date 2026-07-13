package bitbucket

import (
	"fmt"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
)

var hooksDescription = "Stormkit Deploy Hook"
var hooksPath = "/app/webhooks/bitbucket"

// WebhooksRequest represents a webhooks request.
type WebhooksRequest struct {
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Active      bool     `json:"active"`
	Events      []string `json:"events"`
}

// WebhooksResponse represents a webhooks response payload.
type WebhooksResponse struct {
	Values []struct {
		URL string `json:"url"`
	} `json:"values"`
}

// InstallWebhooks install webhooks for the given repository.
//
// A successful operation will return an empty error.
// Error 409: Hooks are already installed
// Error 403: Either access token expired or credentials have not enough permissions.
func (b *Bitbucket) InstallWebhooks(a *App) error {
	if err := b.MustDeployKey(a); err != nil {
		return err
	}

	if b.isHookInstalled(a) {
		return nil
	}

	cnf := admin.MustConfig()

	_, err := b.post(b.hooksEndpoint(a), WebhooksRequest{
		Description: hooksDescription,
		Active:      true,
		URL:         cnf.ApiURL(fmt.Sprintf(hooksPath+"/%s", a.Secret)),
		Events: []string{
			"repo:push",
			"pullrequest:created",
			"pullrequest:fulfilled",
		},
	})

	return err
}

// hooksInstalled checks if the hooks are already installed or not.
func (b *Bitbucket) isHookInstalled(a *App) bool {
	response, err := b.get(b.hooksEndpoint(a))

	if err != nil {
		return false
	}

	hooks := WebhooksResponse{}

	if err := b.parse(response, &hooks); err != nil {
		return false
	}

	for _, val := range hooks.Values {
		if strings.HasPrefix(val.URL, admin.MustConfig().ApiURL(hooksPath)) {
			return true
		}
	}

	return false
}

// hooksEndpoint returns the hooks endpoint.
func (b *Bitbucket) hooksEndpoint(a *App) string {
	owner, repo := oauth.ParseRepo(a.Repo)
	return fmt.Sprintf("/repositories/%s/%s/hooks", owner, repo)
}
