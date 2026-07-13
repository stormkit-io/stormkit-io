package app

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"
)

// Error strings
var (
	ErrNoConnection            = shttperr.New(http.StatusServiceUnavailable, "Provider is not reachable.", "no-connection")
	ErrProviderNotConnected    = shttperr.New(http.StatusForbidden, "Repository is not connected.", "repo-not-connected")
	ErrRepoInvalidProvider     = shttperr.New(http.StatusBadRequest, "Provider is not supported. Supported providers are Github, Bitbucket and Gitlab.", "invalid-provider")
	ErrRepoInvalidFormat       = shttperr.New(http.StatusBadRequest, "Excepting repo to be in :owner/:slug format.", "invalid-repo-format")
	ErrInvalidAppSecret        = shttperr.New(http.StatusBadRequest, "App secret cannot be decyphered or not found. Please re-install app.", "invalid-app-secret")
	ErrInvalidStormkitFile     = shttperr.New(http.StatusBadRequest, "Please make sure that stormkit.config.yml file is valid.", "invalid-stormkit-file")
	ErrMissingOrInvalidAppID   = shttperr.New(http.StatusBadRequest, "This request needs to include the `appId` string value to the request body.", "invalid-app-id")
	ErrInvalidDisplayName      = shttperr.New(http.StatusBadRequest, "The display name can only contain alphanumeric characters, hyphens (-) and cannot be empty.", "invalid-display-name")
	ErrDoubleHyphenDisplayName = shttperr.New(http.StatusBadRequest, "Double hyphens (--) are not allowed as they are reserved for Stormkit.", "invalid-display-name")
	ErrDuplicateDisplayName    = shttperr.New(http.StatusBadRequest, "The display name is already in use. Please choose another one.", "duplicate-display-name")
	ErrEnvironmentNameNotFound = shttperr.New(http.StatusBadRequest, "The environment name does not exist", "env-name-not-found")
	ErrInvalidRuntime          = shttperr.New(http.StatusBadRequest, "The specified runtime is not supported", "invalid-runtime")
	ErrCannotUpdateRuntime     = shttperr.New(http.StatusBadGateway, "Updating runtime failed due to an internal error. Please retry again and reach hello@stormkit.io if the problem persists.", "update-runtime-error")
	ErrInvalidAutoDeployValue  = shttperr.New(http.StatusBadRequest, "Auto deploy can take one of the following values: disabled, commit, pull_request.", "invalid-auto-deploy")
	ErrInvalidWebhookURL       = shttperr.New(http.StatusBadRequest, "The Webhook URL is not valid", "invalid-webhook")
	ErrPaymentRequired         = shttperr.New(http.StatusPaymentRequired, "You need to upgrade before creating new applications", "upgrade-required")
	ErrMissingEnv              = shttperr.New(http.StatusBadRequest, "Environment does not exist.", "missing-env")
)
