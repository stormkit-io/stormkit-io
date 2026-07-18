package oauth

import (
	"golang.org/x/oauth2"
)

// Repo represents a provider repository
type Repo struct {
	URL         string `json:"cloneUrl"`
	Name        string `json:"name"`
	FullName    string `json:"fullName"`
	Description string `json:"description"`
	Provider    string `json:"provider"`
	Private     bool   `json:"private"`
}

type Email struct {
	Address    string `json:"address"`
	IsPrimary  bool   `json:"primary"`
	IsVerified bool   `json:"verified"`
}

// User represents a user returned from the providers.
type User struct {
	*oauth2.Token

	AccountURI   string
	AvatarURI    string
	Emails       []Email
	DisplayName  string
	FullName     string
	ProviderName string
	IsAdmin      bool
}

// NewAdminUser builds the User passed to MustUser when provisioning the
// initial admin, whose single email is treated as primary and verified.
func NewAdminUser(email string) *User {
	return &User{
		Emails:  []Email{{Address: email, IsPrimary: true, IsVerified: true}},
		IsAdmin: true,
	}
}
