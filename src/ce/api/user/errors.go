package user

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"
)

// Error codes
var (
	ErrInvalidRepoFormat           = shttperr.New(http.StatusBadRequest, "Repository name format is not correct. Please re-install app.", "invalid-repo")
	ErrAccessTokenExpired          = shttperr.New(http.StatusForbidden, "Access token expired. Please reconnect your provider.", "token-expired")
	ErrCredsInvalidPermissions     = shttperr.New(http.StatusForbidden, "The credentials lack one or more required privilege scopes to install webhooks.", "invalid-credentials")
	ErrHooksAlreadyInstalled       = shttperr.New(http.StatusConflict, "Webhooks are already installed.", "webhooks-conflict")
	ErrUserAlreadyInTheSystem      = shttperr.New(http.StatusConflict, "The invited user is already an active Stormkit user.", "user-already-member")
	ErrInvalidDisplayName          = shttperr.New(http.StatusBadRequest, "The display name is a required field.", "display-name-missing")
	ErrEmailMissing                = shttperr.New(http.StatusBadRequest, "The email is a required field.", "email-missing")
	ErrDisplayNameMissing          = shttperr.New(http.StatusBadRequest, "The username is a required field.", "display-name-missing")
	ErrAlreadySubscribed           = shttperr.New(http.StatusConflict, "You are already subscribed", "already-subscribed")
	ErrInvalidEmail                = shttperr.New(http.StatusBadRequest, "The email is a not valid.", "email-invalid")
	ErrInvalidProvider             = shttperr.New(http.StatusBadRequest, "Either the provider name is missing or is invalid one. Currently valid values are: github and bitbucket.", "err-invalid-provider")
	ErrSubscriptionInvalidName     = shttperr.New(http.StatusBadRequest, "The subscription name is invalid. It can be one of: free, starter, medium, enterprise.", "invalid-subscription")
	ErrSubscriptionInvalidRemove   = shttperr.New(http.StatusBadRequest, "The card id is a required field.", "invalid-card-id")
	ErrSubscriptionDowngradeFail   = shttperr.New(http.StatusUnauthorized, "Requested plan is too small for current usage. Check %s for more information on packages.", "downgrade-not-allowed")
	ErrSubscriptionNoPaymentMethod = shttperr.New(http.StatusPaymentRequired, "User has no payment method attached", "no-payment-method")
)
