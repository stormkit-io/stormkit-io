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
