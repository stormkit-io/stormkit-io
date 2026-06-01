package hosting

import (
	"fmt"
	"net/http"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"golang.org/x/crypto/bcrypt"
)

// emlKey derives a 32-byte AES key for encrypting the `eml` JWT claim.
// SKAuth secrets are normally 128-char ASCII tokens; if a shorter secret
// is encountered (which AES rejects), fall back to the app secret.
func emlKey(secret string) []byte {
	if len(secret) >= 32 {
		return []byte(secret)[:32]
	}

	return []byte(config.AppSecret())
}

type emailAuthCtx struct {
	env      *buildconf.Env
	envID    types.ID
	email    string
	password string
}

// resolveEmailAuth parses the request body and resolves the environment and email
// provider, returning a context shared by the login and register handlers.
// The environment ID is always taken from the host config, never from the request,
// to prevent cross-tenant confusion.
func (m *skAuthMiddleware) resolveEmailAuth() (*emailAuthCtx, *shttp.Response) {
	req := m.req.RequestContext

	body := &struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{}

	if err := req.Post(body); err != nil {
		return nil, shttp.BadRequest(map[string]any{"errors": []string{err.Error()}})
	}

	envID := m.req.Host.Config.EnvID
	email := body.Email
	password := body.Password

	if envID == 0 {
		return nil, shttp.NotFound()
	}

	if errs := validateEmailAuth(email, password); len(errs) > 0 {
		return nil, shttp.BadRequest(map[string]any{"errors": errs})
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), envID)

	if err != nil {
		return nil, shttp.Error(err, fmt.Sprintf("failed to get environment by ID %d", envID))
	}

	if env == nil || env.AuthConf == nil || !env.AuthConf.Status || env.SchemaConf == nil {
		return nil, shttp.NotFound()
	}

	prv, err := skauth.NewStore().Provider(req.Context(), envID, skauth.ProviderEmail)

	if err != nil {
		return nil, shttp.Error(err, fmt.Sprintf("failed to get provider: %s", err.Error()))
	}

	if prv == nil || !prv.Status {
		return nil, shttp.NotFound()
	}

	return &emailAuthCtx{env: env, envID: envID, email: email, password: password}, nil
}

func (m *skAuthMiddleware) login() *shttp.Response {
	ctx, errResp := m.resolveEmailAuth()

	if errResp != nil {
		return errResp
	}

	req := m.req.RequestContext

	store, err := ctx.env.SchemaConf.Store(buildconf.SchemaAccessTypeAppUser)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get schema store: %s", err.Error()))
	}

	authUser, err := store.AuthUserByEmail(req.Context(), buildconf.AuthUserByEmailParams{
		Email:    ctx.email,
		Provider: skauth.ProviderEmail,
	})

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to fetch user: %s", err.Error()))
	}

	if authUser == nil || authUser.PasswordHash == "" {
		return shttp.BadRequest(map[string]any{"errors": []string{"invalid email or password"}})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(authUser.PasswordHash), []byte(ctx.password)); err != nil {
		return shttp.BadRequest(map[string]any{"errors": []string{"invalid email or password"}})
	}

	if ctx.env.MailerConf != nil && authUser.VerifiedAt.IsZero() {
		return &shttp.Response{
			Status: http.StatusForbidden,
			Data:   map[string]any{"errors": []string{"please verify your email address before logging in"}},
		}
	}

	sessionToken, err := user.JWT(jwt.MapClaims{
		"uid": authUser.UUID,
		"eml": utils.EncryptToString(ctx.email, emlKey(ctx.env.AuthConf.Secret)),
		"eid": fmt.Sprintf("%d", ctx.envID),
		"prv": skauth.ProviderEmail,
	}, ctx.env.AuthConf.Secret)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to generate session token: %s", err.Error()))
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   map[string]any{"token": sessionToken, "email": ctx.email, "userId": authUser.UUID},
	}
}

func (m *skAuthMiddleware) register() *shttp.Response {
	ctx, errResp := m.resolveEmailAuth()

	if errResp != nil {
		return errResp
	}

	req := m.req.RequestContext

	hash, err := bcrypt.GenerateFromPassword([]byte(ctx.password), bcrypt.DefaultCost)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to hash password: %s", err.Error()))
	}

	store, err := ctx.env.SchemaConf.Store(buildconf.SchemaAccessTypeAppUser)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get schema store: %s", err.Error()))
	}

	oauth := skauth.OAuth{
		AccountID:    ctx.email,
		AccessToken:  string(hash),
		TokenType:    "password",
		ProviderName: skauth.ProviderEmail,
	}

	if ctx.env.MailerConf != nil {
		verificationToken, err := generateEmailVerificationToken(ctx.env)

		if err != nil {
			return shttp.Error(err, fmt.Sprintf("failed to generate verification token: %s", err.Error()))
		}

		oauth.VerificationToken = verificationToken
	}

	usr := skauth.User{
		Email: ctx.email,
	}

	if err := store.InsertAuthUser(req.Context(), &oauth, &usr); err != nil {
		if database.IsDuplicate(err) {
			return shttp.BadRequest(map[string]any{"errors": []string{"an account with this email already exists"}})
		}

		return shttp.Error(err, fmt.Sprintf("failed to register user: %s", err.Error()))
	}

	if ctx.env.MailerConf != nil {
		if err := sendVerificationEmail(req, ctx.env, ctx.email, oauth.VerificationToken); err != nil {
			return shttp.Error(err, fmt.Sprintf("failed to send verification email: %s", err.Error()))
		}

		return &shttp.Response{
			Status: http.StatusCreated,
			Data:   map[string]any{"verificationRequired": true, "email": ctx.email, "userId": usr.UUID},
		}
	}

	sessionToken, err := user.JWT(jwt.MapClaims{
		"uid": usr.UUID,
		"eml": utils.EncryptToString(ctx.email, emlKey(ctx.env.AuthConf.Secret)),
		"eid": fmt.Sprintf("%d", ctx.envID),
		"prv": skauth.ProviderEmail,
	}, ctx.env.AuthConf.Secret)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to generate session token: %s", err.Error()))
	}

	return &shttp.Response{
		Status: http.StatusCreated,
		Data:   map[string]any{"token": sessionToken, "email": ctx.email, "userId": usr.UUID},
	}
}

func (m *skAuthMiddleware) verifyEmail() *shttp.Response {
	req := m.req.RequestContext

	token := req.Query().Get("token")

	if token == "" {
		return shttp.BadRequest(map[string]any{"errors": []string{"token is required"}})
	}

	envID := m.req.Host.Config.EnvID

	if envID == 0 {
		return shttp.NotFound()
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

	if claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: token, Secret: env.AuthConf.Secret}); claims == nil {
		return shttp.BadRequest(map[string]any{"errors": []string{"invalid or expired verification token"}})
	}

	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeAppUser)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get schema store: %s", err.Error()))
	}

	userID, err := store.VerifyEmailUser(req.Context(), token)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to verify email: %s", err.Error()))
	}

	if userID == 0 {
		return shttp.BadRequest(map[string]any{"errors": []string{"invalid or expired verification token"}})
	}

	authUser, err := store.AuthUser(req.Context(), userID)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to fetch user: %s", err.Error()))
	}

	if authUser == nil {
		return shttp.Error(fmt.Errorf("user %d not found after verification", userID), "internal error")
	}

	sessionToken, err := user.JWT(jwt.MapClaims{
		"uid": authUser.UUID,
		"eml": utils.EncryptToString(authUser.Email, emlKey(env.AuthConf.Secret)),
		"eid": fmt.Sprintf("%d", envID),
		"prv": skauth.ProviderEmail,
	}, env.AuthConf.Secret)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to generate session token: %s", err.Error()))
	}

	return &shttp.Response{
		Data: map[string]any{"token": sessionToken, "email": authUser.Email, "userId": authUser.UUID},
	}
}

func validateEmailAuth(email, password string) []string {
	errs := []string{}

	if !utils.IsValidEmail(email) {
		errs = append(errs, "email is invalid")
	}

	if len(password) < 8 {
		errs = append(errs, "password must be at least 8 characters")
	}

	return errs
}

func generateEmailVerificationToken(env *buildconf.Env) (string, error) {
	jti, err := utils.SecureRandomToken(16)

	if err != nil {
		return "", err
	}

	return user.JWT(jwt.MapClaims{
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"prv": skauth.ProviderEmail,
		"jti": jti,
	}, env.AuthConf.Secret)
}

func sendVerificationEmail(req *shttp.RequestContext, env *buildconf.Env, email, token string) error {
	u := req.URL()
	verifyLink := fmt.Sprintf("%s://%s/_stormkit/auth/verify?token=%s", u.Scheme, u.Host, token)

	return env.MailerConf.Send(buildconf.SendEmailParams{
		To:      email,
		Subject: "Verify your email address",
		Body:    fmt.Sprintf(`<p>Click the link below to verify your email address:</p><p><a href="%s">%s</a></p>`, verifyLink, verifyLink),
	})
}
