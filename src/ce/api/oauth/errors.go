package oauth

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"
)

// Error codes
var (
	ErrNotAuthorized           = shttperr.New(401, "User is not authorised to perform action. Make sure to have appropriate permissions.", "not-authorized")
	ErrInvalidRepoFormat       = shttperr.New(http.StatusBadRequest, "Repository name format is not correct. Please re-install app.", "invalid-repo")
	ErrInvalidProvider         = shttperr.New(http.StatusBadRequest, "Provider is not supported. Supported providers are Github, Bitbucket and Gitlab.", "invalid-provider")
	ErrCredsInvalidPermissions = shttperr.New(http.StatusForbidden, "Either access token expired or credentials lack one or more required privilege scopes. Please reconnect your provider.", "invalid-credentials")
	ErrRepoNotFound            = shttperr.New(http.StatusNotFound, "Repository is not accessible.", "repo-not-found")
	ErrBranchNotFound          = shttperr.New(http.StatusNotFound, "Branch not found", "branch-not-found")
	ErrHooksAlreadyInstalled   = shttperr.New(http.StatusConflict, "Webhooks are already installed.", "webhooks-conflict")
	ErrInvalidResponseCode     = shttperr.New(http.StatusExpectationFailed, "Invalid response from provider.", "invalid-response")
	ErrKeyAlreadyInstalled     = shttperr.New(http.StatusConflict, "Deploy key has been already installed", "deploy-key-installed")
	ErrProviderNotReachable    = shttperr.New(http.StatusServiceUnavailable, "Git provider is not reachable for the moment. Please check their status page and retry later.", "provider-unreachable")
	ErrProviderNotConnected    = shttperr.New(http.StatusUnauthorized, "Provider is not connected.", "provider-not-connected")
	ErrGithubPrivateKeyInvalid = shttperr.New(http.StatusInternalServerError, "Error while signing jwt token", "github-jwt")
	ErrGithubAppNotFound       = shttperr.New(http.StatusNotFound, "Stormkit Github App not found on this repository.", "github-app-not-found")
)
