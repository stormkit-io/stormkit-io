package bitbucket

import (
	"errors"

	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
)

// EmailsResponse represents the email list response.
type EmailsResponse struct {
	PageLen int `json:"pagelen"`
	Size    int `json:"size"`
	Values  []*struct {
		Email    string `json:"email"`
		Primary  bool   `json:"is_primary"`
		Verified bool   `json:"is_confirmed"`
		Type     string `json:"type"`
		Links    struct {
			Self struct {
				Href string `json:"href"`
			}
		} `json:"links"`
	}
}

// ProfileResponse represents a user profile response.
type ProfileResponse struct {
	UserName    string `json:"username"`     // This is the username which is unique and will not change
	NickName    string `json:"nickname"`     // This is the nickname the user wants to display to others
	DisplayName string `json:"display_name"` // This is the full name
	Links       struct {
		HTML      LinkHref `json:"html"`
		AvatarURI LinkHref `json:"avatar"`
	} `json:"links"`
}

// UserInterface represents all requests related to user.
type UserInterface struct {
	b *Bitbucket
}

// PrimaryEmail returns the primary email for the user.
func (ui *UserInterface) PrimaryEmail() (string, error) {
	response, err := ui.b.get("/user/emails")

	if err != nil {
		return "", err
	}

	emails := &EmailsResponse{}

	if err := ui.b.parse(response, emails); err != nil {
		return "", nil
	}

	if emails.Values != nil {
		for _, record := range emails.Values {
			if record.Primary && record.Verified {
				return record.Email, nil
			}
		}
	}

	return "", errors.New("cannot find primary email")
}

// Profile fetches the user profile information and returns it.
// This endpoint returns the authenticated user.
func (ui *UserInterface) Profile() (*oauth.User, error) {
	response, err := ui.b.get("/user")

	if err != nil {
		return nil, err
	}

	login := &ProfileResponse{}

	if err := ui.b.parse(response, login); err != nil {
		return nil, err
	}

	ui.b.user.AccountURI = login.Links.HTML.Href
	ui.b.user.AvatarURI = login.Links.AvatarURI.Href
	ui.b.user.DisplayName = login.NickName
	ui.b.user.FullName = login.DisplayName

	primaryEmail, err := ui.PrimaryEmail()

	if primaryEmail != "" {
		ui.b.user.Emails = []oauth.Email{
			{Address: primaryEmail, IsPrimary: true, IsVerified: true},
		}
	}

	return ui.b.user, err
}
