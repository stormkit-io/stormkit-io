package hosting

import (
	"fmt"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func (m *skAuthMiddleware) magicLinkRequest() *shttp.Response {
	req := m.req.RequestContext

	body := &struct {
		Email    string `json:"email"`
		Redirect string `json:"redirect"`
	}{}

	_ = req.Post(body)

	envID := m.req.Host.Config.EnvID
	email := utils.GetString(body.Email, req.Query().Get("email"))

	if envID == 0 {
		return shttp.NotFound()
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

	// redirect is where the user is sent back to after clicking the email link.
	// Browsers may omit it (we fall back to the Origin header); native clients
	// pass it explicitly (e.g. a deep link). It is honoured only when it is on
	// the configured allow-list; otherwise the flow stays single-host.
	redirect := utils.GetString(body.Redirect, req.Query().Get("redirect"))

	if redirect == "" {
		redirect = m.req.Header.Get("Origin")
	}

	redirect = strings.TrimRight(redirect, "/")

	// When an allow-list is configured, a supplied redirect must be on it.
	if len(env.AuthConf.AllowedOrigins) > 0 && redirect != "" && !env.AuthConf.IsAllowedOrigin(redirect) {
		return shttp.Forbidden()
	}

	if !env.AuthConf.IsAllowedOrigin(redirect) {
		redirect = ""
	}

	prv, err := skauth.NewStore().Provider(req.Context(), envID, skauth.ProviderMagicLink)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get provider: %s", err.Error()))
	}

	if prv == nil || !prv.Status {
		return shttp.NotFound()
	}

	token, err := generateMagicLinkToken(env, redirect)

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

	if err := sendMagicLinkEmail(req, env, prv, email, token); err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to send magic link email: %s", err.Error()))
	}

	return shttp.OK()
}

func (m *skAuthMiddleware) magicLinkVerify() *shttp.Response {
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

	prv, err := skauth.NewStore().Provider(req.Context(), envID, skauth.ProviderMagicLink)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get provider: %s", err.Error()))
	}

	if prv == nil || !prv.Status {
		return shttp.NotFound()
	}

	claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: token, Secret: env.AuthConf.Secret})

	if claims == nil {
		return shttp.BadRequest(map[string]any{"errors": []string{"invalid or expired magic link token"}})
	}

	redirectOrigin, _ := claims["rdr"].(string)

	// Re-validate at click time in case the allow-list changed between
	// request and verify; don't trust a stale claim.
	if redirectOrigin != "" && !env.AuthConf.IsAllowedOrigin(redirectOrigin) {
		redirectOrigin = ""
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
		"uid": authUser.UUID,
		"eml": utils.EncryptToString(authUser.Email, emlKey(env.AuthConf.Secret)),
		"eid": fmt.Sprintf("%d", envID),
		"prv": skauth.ProviderMagicLink,
	}, env.AuthConf.Secret)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to generate session token: %s", err.Error()))
	}

	data := map[string]any{"token": sessionToken, "email": authUser.Email, "userId": authUser.UUID}

	if redirectOrigin != "" {
		data["redirect"] = redirectOrigin
	}

	return &shttp.Response{Data: data}
}

func generateMagicLinkToken(env *buildconf.Env, redirectOrigin string) (string, error) {
	jti, err := utils.SecureRandomToken(16)

	if err != nil {
		return "", err
	}

	claims := jwt.MapClaims{
		"exp": time.Now().Add(15 * time.Minute).Unix(),
		"prv": skauth.ProviderMagicLink,
		"jti": jti,
	}

	if redirectOrigin != "" {
		claims["rdr"] = redirectOrigin
	}

	return user.JWT(claims, env.AuthConf.Secret)
}

func sendMagicLinkEmail(req *shttp.RequestContext, env *buildconf.Env, prv *skauth.Provider, email, token string) error {
	u := req.URL()
	link := fmt.Sprintf("%s://%s/_stormkit/auth/magic?token=%s", u.Scheme, u.Host, token)

	params := buildconf.SendEmailParams{
		To:      email,
		From:    prv.Data.FromAddress,
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
