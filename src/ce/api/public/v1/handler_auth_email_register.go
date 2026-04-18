package publicapiv1

import (
	"fmt"
	"net/http"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"golang.org/x/crypto/bcrypt"
)

// HandlerAuthEmailRegister registers a new user with email and password.
// On success it returns a JSON response with a session token:
//
//	{"token": "<jwt>"}
//
// On failure it returns a JSON 400 with an "errors" key.
// The handler is registered at POST /v1/auth/register and is also
// callable from the hosting layer via /_stormkit/auth/register (WithSKAuth).
// POST /v1/auth/register
func HandlerAuthEmailRegister(req *shttp.RequestContext) *shttp.Response {
	body := &struct {
		EnvID    string `json:"envId"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}{}

	if err := req.Post(body); err != nil {
		return shttp.BadRequest(map[string]any{"errors": []string{err.Error()}})
	}

	// envId can be supplied in the JSON body or injected into the URL query
	// string by the hosting middleware (/_stormkit/auth/register path).
	envIDStr := body.EnvID

	if envIDStr == "" {
		envIDStr = req.FormValue("envId")
	}

	envID := utils.StringToID(envIDStr)
	email := body.Email
	password := body.Password

	if envID == 0 {
		return shttp.BadRequest(map[string]any{"errors": []string{"envId is required"}})
	}

	if errs := validateEmailAuthRequest(email, password); len(errs) > 0 {
		return shttp.BadRequest(map[string]any{"errors": errs})
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
			return shttp.BadRequest(map[string]any{"errors": []string{"an account with this email already exists"}})
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

	return &shttp.Response{
		Status: http.StatusCreated,
		Data:   map[string]any{"token": sessionToken, "email": email, "userId": usr.ID},
	}
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
