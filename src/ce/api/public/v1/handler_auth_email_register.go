package publicapiv1

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"golang.org/x/crypto/bcrypt"
)

// HandlerAuthEmailRegister registers a new user with email and password.
// It accepts a form POST and on success redirects to /_stormkit/auth?code=X,
// where the hosting layer (WithSKAuth) exchanges the code for a session token
// and redirects to the configured SuccessURL.
// On failure: returns JSON 400 when the env is unknown or the Referer cannot
// be validated; redirects to the Referer with ?error=<message> once the env
// is resolved and the Referer host is confirmed to belong to it.
// POST /v1/auth/register
func HandlerAuthEmailRegister(req *shttp.RequestContext) *shttp.Response {
	envID := utils.StringToID(req.FormValue("envId"))
	email := req.FormValue("email")
	password := req.FormValue("password")

	if envID == 0 {
		return redirectAuthError(req, nil, "envId is required")
	}

	if errs := validateEmailAuthRequest(email, password); len(errs) > 0 {
		return redirectAuthError(req, nil, errs[0])
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), envID)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get environment by ID %d", envID))
	}

	if env == nil || env.AuthConf == nil || !env.AuthConf.Status || env.SchemaConf == nil {
		return shttp.NotFound()
	}

	prv, err := skauth.NewStore().Provider(req.Context(), envID, skauth.ProviderEmail)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get provider: %s", err.Error()))
	}

	if prv == nil || !prv.Status {
		return shttp.NotFound()
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to hash password: %s", err.Error()))
	}

	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeAppUser)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get schema store: %s", err.Error()))
	}

	oauth := skauth.OAuth{
		AccountID:    email,
		AccessToken:  string(hash),
		TokenType:    "password",
		ProviderName: skauth.ProviderEmail,
	}

	usr := skauth.User{
		Email: email,
	}

	if err := store.InsertAuthUser(req.Context(), &oauth, &usr); err != nil {
		if database.IsDuplicate(err) {
			return redirectAuthError(req, env, "an account with this email already exists")
		}

		return shttp.Error(err, fmt.Sprintf("failed to register user: %s", err.Error()))
	}

	sessionToken, err := user.JWT(jwt.MapClaims{
		"uid": usr.ID,
		"eid": fmt.Sprintf("%d", envID),
		"prv": skauth.ProviderEmail,
	}, env.AuthConf.Secret)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to generate session token: %s", err.Error()))
	}

	code, err := utils.SecureRandomToken(32)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to generate auth code: %s", err.Error()))
	}

	if err := rediscache.Client().Set(req.Context(), code, sessionToken, time.Minute*2).Err(); err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to store session token: %s", err.Error()))
	}

	// WithSKAuth (hosting layer) reads SuccessURL from its own config, so there
	// is no need to pass it here — doing so would be redundant and misleading.
	req.Redirect(
		fmt.Sprintf("/_stormkit/auth?code=%s", code),
		http.StatusFound,
	)

	return nil
}

// redirectAuthError redirects back to the HTTP Referer with ?error=<msg>.
// Falls back to a JSON 400 response when no valid Referer header is present,
// when env is nil, or when the Referer host does not belong to the environment
// (to prevent open-redirect attacks).
func redirectAuthError(req *shttp.RequestContext, env *buildconf.Env, errMsg string) *shttp.Response {
	referer := req.Referer()

	if referer == "" || env == nil {
		return shttp.BadRequest(map[string]any{"errors": []string{errMsg}})
	}

	parsed, err := url.ParseRequestURI(referer)

	if err != nil {
		return shttp.BadRequest(map[string]any{"errors": []string{errMsg}})
	}

	hostname := parsed.Hostname()
	belongs, err := appconf.NewStore().BelongsToEnv(req.Context(), env.ID, appconf.ParseHost(hostname))

	if err != nil || !belongs {
		return shttp.BadRequest(map[string]any{"errors": []string{errMsg}})
	}

	q := parsed.Query()
	q.Set("error", errMsg)
	parsed.RawQuery = q.Encode()

	req.Redirect(parsed.String(), http.StatusFound)

	return nil
}

// validateEmailAuthRequest validates email and password fields shared by register and login.
func validateEmailAuthRequest(email, password string) []string {
	errs := []string{}

	if !utils.IsValidEmail(email) {
		errs = append(errs, "email is invalid")
	}

	if len(password) < 8 {
		errs = append(errs, "password must be at least 8 characters")
	}

	return errs
}
