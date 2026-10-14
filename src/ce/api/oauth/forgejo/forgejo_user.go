package forgejo

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
)

type apiUser struct {
	Login     string `json:"login"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
}

type apiEmail struct {
	Email    string `json:"email"`
	Verified bool   `json:"verified"`
	Primary  bool   `json:"primary"`
}

// UserProfile returns information on the user.
func (f *Forgejo) UserProfile() (*oauth.User, error) {
	usr := apiUser{}

	if _, err := f.do(request{method: http.MethodGet, path: "/user", out: &usr}); err != nil {
		return nil, err
	}

	f.user.AvatarURI = usr.AvatarURL
	f.user.AccountURI = usr.HTMLURL
	f.user.DisplayName = usr.Login
	f.user.FullName = usr.FullName

	emails := []apiEmail{}

	if _, err := f.do(request{method: http.MethodGet, path: "/user/emails", out: &emails}); err != nil {
		return nil, err
	}

	f.user.Emails = []oauth.Email{}

	// Sign-in matches existing accounts by email, and anyone can add an
	// address to their Forgejo account without confirming it.
	for _, email := range emails {
		if !email.Verified {
			continue
		}

		f.user.Emails = append(f.user.Emails, oauth.Email{
			Address:    email.Email,
			IsPrimary:  email.Primary,
			IsVerified: email.Verified,
		})
	}

	return f.user, nil
}
