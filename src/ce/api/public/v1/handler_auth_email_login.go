package publicapiv1

import (
	"fmt"
	"net/http"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"golang.org/x/crypto/bcrypt"
)

// HandlerAuthEmailLogin authenticates an existing user with email and password.
// On success it returns a JSON response with a session token:
//
//	{"token": "<jwt>"}
//
// POST /v1/auth/login
func HandlerAuthEmailLogin(req *shttp.RequestContext) *shttp.Response {
	body := &struct {
		EnvID    string `json:"envId"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}{}

	if err := req.Post(body); err != nil {
		return shttp.BadRequest(map[string]any{"errors": []string{err.Error()}})
	}

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

	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeAppUser)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get schema store: %s", err.Error()))
	}

	authUser, err := store.AuthUserByEmail(req.Context(), buildconf.AuthUserByEmailParams{
		Email:    email,
		Provider: skauth.ProviderEmail,
	})

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to fetch user: %s", err.Error()))
	}

	if authUser == nil || authUser.PasswordHash == "" {
		return shttp.BadRequest(map[string]any{"errors": []string{"invalid email or password"}})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(authUser.PasswordHash), []byte(password)); err != nil {
		return shttp.BadRequest(map[string]any{"errors": []string{"invalid email or password"}})
	}

	if env.MailerConf != nil && authUser.VerifiedAt.IsZero() {
		return &shttp.Response{
			Status: http.StatusForbidden,
			Data:   map[string]any{"errors": []string{"please verify your email address before logging in"}},
		}
	}

	sessionToken, err := user.JWT(jwt.MapClaims{
		"uid": authUser.ID,
		"eid": fmt.Sprintf("%d", envID),
		"prv": skauth.ProviderEmail,
	}, env.AuthConf.Secret)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to generate session token: %s", err.Error()))
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   map[string]any{"token": sessionToken, "email": email, "userId": authUser.ID},
	}
}
