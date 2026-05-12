package publicapiv1

import (
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func generateMagicLinkToken(env *buildconf.Env) (string, error) {
	jti, err := utils.SecureRandomToken(16)

	if err != nil {
		return "", err
	}

	return user.JWT(jwt.MapClaims{
		"exp": time.Now().Add(15 * time.Minute).Unix(),
		"prv": skauth.ProviderMagicLink,
		"jti": jti,
	}, env.AuthConf.Secret)
}

func sendMagicLinkEmail(req *shttp.RequestContext, env *buildconf.Env, email, token string) error {
	u := req.URL()
	link := fmt.Sprintf("%s://%s/_stormkit/auth/magic?token=%s", u.Scheme, u.Host, token)

	params := buildconf.SendEmailParams{
		To:      email,
		Subject: "Your magic link",
		Body:    fmt.Sprintf(`<p>Click the link below to sign in. The link expires in 15 minutes.</p><p><a href="%s">%s</a></p>`, link, link),
	}

	if err := buildconf.MailerStore().InsertEmail(req.Context(), buildconf.Email{
		EnvID:   env.ID,
		To:      email,
		Subject: params.Subject,
		Body:    params.Body,
	}); err != nil {
		return err
	}

	if env.MailerConf != nil {
		return env.MailerConf.Send(params)
	}

	return nil
}

// HandlerAuthMagicLinkRequest handles a magic link sign-in request.
// It finds or creates the user, stores a short-lived token, and emails the link.
// POST /v1/auth/magiclink
func HandlerAuthMagicLinkRequest(req *shttp.RequestContext) *shttp.Response {
	body := &struct {
		EnvID string `json:"envId"`
		Email string `json:"email"`
	}{}

	// Ignore parse errors — the fields may arrive as query params instead of a JSON body.
	_ = req.Post(body)

	envID := utils.StringToID(utils.GetString(body.EnvID, req.FormValue("envId"), req.Query().Get("envId")))
	email := utils.GetString(body.Email, req.Query().Get("email"))

	if envID == 0 {
		return shttp.BadRequest(map[string]any{"errors": []string{"envId is required"}})
	}

	if !utils.IsValidEmail(email) {
		return shttp.BadRequest(map[string]any{"errors": []string{"email is invalid"}})
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), envID)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get environment by ID %d", envID))
	}

	if env == nil || env.AuthConf == nil || !env.AuthConf.Status || env.SchemaConf == nil {
		return shttp.NotFound()
	}

	prv, err := skauth.NewStore().Provider(req.Context(), envID, skauth.ProviderMagicLink)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get provider: %s", err.Error()))
	}

	if prv == nil || !prv.Status {
		return shttp.NotFound()
	}

	token, err := generateMagicLinkToken(env)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to generate magic link token: %s", err.Error()))
	}

	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeAppUser)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get schema store: %s", err.Error()))
	}

	if _, err := store.UpsertMagicLinkUser(req.Context(), email, token); err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to upsert magic link user: %s", err.Error()))
	}

	if err := sendMagicLinkEmail(req, env, email, token); err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to send magic link email: %s", err.Error()))
	}

	return shttp.OK()
}

// HandlerAuthMagicLinkVerify validates a magic link token and returns a session.
// GET /v1/auth/magiclink?token=&envId=
func HandlerAuthMagicLinkVerify(req *shttp.RequestContext) *shttp.Response {
	token := req.Query().Get("token")
	envIDStr := req.Query().Get("envId")

	if token == "" {
		return shttp.BadRequest(map[string]any{"errors": []string{"token is required"}})
	}

	envID := utils.StringToID(envIDStr)

	if envID == 0 {
		return shttp.BadRequest(map[string]any{"errors": []string{"envId is required"}})
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), envID)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get environment by ID %d", envID))
	}

	if env == nil || env.AuthConf == nil || !env.AuthConf.Status || env.SchemaConf == nil {
		return shttp.NotFound()
	}

	prv, err := skauth.NewStore().Provider(req.Context(), envID, skauth.ProviderMagicLink)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get provider: %s", err.Error()))
	}

	if prv == nil || !prv.Status {
		return shttp.NotFound()
	}

	if claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: token, Secret: env.AuthConf.Secret}); claims == nil {
		return shttp.BadRequest(map[string]any{"errors": []string{"invalid or expired magic link token"}})
	}

	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeAppUser)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get schema store: %s", err.Error()))
	}

	userID, err := store.ConsumeMagicLinkToken(req.Context(), token)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to consume magic link token: %s", err.Error()))
	}

	if userID == 0 {
		return shttp.BadRequest(map[string]any{"errors": []string{"invalid or expired magic link token"}})
	}

	authUser, err := store.AuthUser(req.Context(), userID)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to fetch user: %s", err.Error()))
	}

	if authUser == nil {
		return shttp.Error(fmt.Errorf("user %d not found after consuming magic link", userID), "internal error")
	}

	sessionToken, err := user.JWT(jwt.MapClaims{
		"uid": authUser.ID,
		"eid": fmt.Sprintf("%d", envID),
		"prv": skauth.ProviderMagicLink,
	}, env.AuthConf.Secret)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to generate session token: %s", err.Error()))
	}

	return &shttp.Response{
		Data: map[string]any{"token": sessionToken, "email": authUser.Email, "userId": authUser.ID},
	}
}
