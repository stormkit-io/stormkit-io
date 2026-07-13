package gitlab

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
)

// UserProfile returns information on the user.
func (g *Gitlab) UserProfile() (*oauth.User, error) {
	usr, _, err := g.Users.CurrentUser()

	if err != nil {
		return nil, err
	}

	g.user.AvatarURI = usr.AvatarURL
	g.user.DisplayName = usr.Username
	g.user.FullName = usr.Name

	// TODO: list all emails
	g.user.Emails = []oauth.Email{
		{Address: usr.Email, IsPrimary: true, IsVerified: true},
	}

	return g.user, err
}
