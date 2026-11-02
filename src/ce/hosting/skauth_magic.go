package hosting

import (
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type magicLinkRequestBody struct {
	Email               string `json:"email"`
	Redirect            string `json:"redirect"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
	ReturnTo            string `json:"return_to"`
}

// magicLinkRequest sends a magic link. A request carrying return_to was started
// from the app's OAuth page: the link brings the user back there, and an HTML
// form submission is answered with a redirect back to the page rather than JSON.
func (m *skAuthMiddleware) magicLinkRequest() *shttp.Response {
	req := m.req.RequestContext
	body := &magicLinkRequestBody{}

	_ = req.Post(body)

	returnTo := m.oauthReturnTo(utils.GetString(body.ReturnTo, req.Query().Get("return_to")))
	res := m.sendMagicLink(body, returnTo)

	if returnTo == "" || !isFormPost(req.Request) {
		return res
	}

	outcome := oauthSignInLinkSent

	switch {
	case res.Status == http.StatusBadRequest:
		outcome = oauthSignInInvalidEmail
	case res.Status != 0 && res.Status != http.StatusOK:
		outcome = oauthSignInFailed
	}

	return m.backToOAuth(backToOAuthParams{ReturnTo: returnTo, Outcome: outcome})
}

// sendMagicLink validates the request and emails the link. returnTo, when set,
// takes the place of the redirect origin: the user lands back on this host's
// OAuth page.
func (m *skAuthMiddleware) sendMagicLink(body *magicLinkRequestBody, returnTo string) *shttp.Response {
	req := m.req.RequestContext
	envID := m.req.Host.Config.EnvID
	email := normalizeEmail(utils.GetString(body.Email, req.Query().Get("email"), req.PostFormValue("email")))

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
	var redirect, challenge string

	if returnTo == "" {
		redirect = utils.GetString(body.Redirect, req.Query().Get("redirect"))

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

		// A custom-scheme redirect delivers the session through a PKCE-bound code, so
		// the challenge has to be captured now and travel with the link — the user's
		// inbox sits between this request and the redemption.
		challenge = utils.GetString(body.CodeChallenge, req.Query().Get("code_challenge"))

		if resp := validateNativeChallenge(
			redirect,
			challenge,
			utils.GetString(body.CodeChallengeMethod, req.Query().Get("code_challenge_method")),
		); resp != nil {
			return resp
		}
	}

	prv, err := skauth.NewStore().Provider(req.Context(), envID, skauth.ProviderMagicLink)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get provider: %s", err.Error()))
	}

	if prv == nil || !prv.Status {
		return shttp.NotFound()
	}

	token, err := generateMagicLinkToken(generateMagicLinkTokenParams{
		Env:            env,
		RedirectOrigin: redirect,
		CodeChallenge:  challenge,
		ReturnTo:       returnTo,
	})

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

	err = sendMagicLinkEmail(sendMagicLinkEmailParams{
		Req:      req,
		Env:      env,
		Provider: prv,
		Email:    email,
		Token:    token,
		Redirect: redirect,
	})

	if errors.Is(err, errNoMailer) {
		return &shttp.Response{
			Status: http.StatusServiceUnavailable,
			Data: map[string][]string{
				"errors": {"The sign-in email could not be sent. Please try again later."},
			},
		}
	}

	if err != nil {
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

	claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: token, Secret: env.AuthConf.Secret, Purposes: []user.Purpose{user.PurposeSkAuthMagicLink}})

	if claims == nil {
		// Expiry is the ordinary way a magic link fails, and the user is often in a
		// native app's system browser with nowhere else to land. The origin is
		// recovered from the unvalidated claims purely to aim the error redirect —
		// see unverifiedRedirectOrigin for why that is safe.
		return shttp.BadRequest(map[string]any{
			"errors":   []string{"invalid or expired magic link token"},
			"redirect": unverifiedRedirectOrigin(token, env.AuthConf),
			"returnTo": m.unverifiedReturnTo(token),
		})
	}

	redirectOrigin, _ := claims["rdr"].(string)
	rto, _ := claims["rto"].(string)
	returnTo := m.oauthReturnTo(rto)

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
		// We parsed the token, so we know the initiating origin — pass it through
		// so the error redirect lands on the right app rather than this host.
		return shttp.BadRequest(map[string]any{
			"errors":   []string{"invalid or expired magic link token"},
			"redirect": redirectOrigin,
			"returnTo": returnTo,
		})
	}

	authUser, err := store.AuthUser(req.Context(), userID)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to fetch user: %s", err.Error()))
	}

	if authUser == nil {
		return shttp.Error(fmt.Errorf("user %d not found after consuming magic link", userID), "internal error")
	}

	if err := store.UpdateLastLogin(req.Context(), userID); err != nil {
		slog.Errorf("magic link verify: failed to update last login: %s", err.Error())
	}

	sessionToken, err := user.JWT(user.JWTParams{Purpose: user.PurposeSkAuthSession, Claims: jwt.MapClaims{
		"uid": authUser.UUID,
		"eml": utils.EncryptToString(authUser.Email, emlKey(env.AuthConf.Secret)),
		"eid": fmt.Sprintf("%d", envID),
		"prv": skauth.ProviderMagicLink,
	}, Secret: env.AuthConf.Secret})

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to generate session token: %s", err.Error()))
	}

	data := map[string]any{"token": sessionToken, "email": authUser.Email, "userId": authUser.UUID}

	if returnTo != "" {
		data["returnTo"] = returnTo

		return &shttp.Response{Data: data}
	}

	if redirectOrigin != "" {
		data["redirect"] = redirectOrigin

		if challenge, _ := claims["cha"].(string); challenge != "" {
			data["codeChallenge"] = challenge
		}
	}

	return &shttp.Response{Data: data}
}

// unverifiedRedirectOrigin recovers the `rdr` claim from a magic-link token that
// failed validation, so an expired or malformed link can still bounce the user
// back to the app that started the sign-in instead of stranding them on this
// host — which, in a native app's system browser, means a sheet that never
// closes.
//
// Reading unvalidated claims is safe here because the signature is not what
// authorizes the target: the recovered origin is re-checked against the
// environment's allow-list, so a forged claim can only ever name an origin the
// operator already registered. It is used solely to aim an error redirect; no
// session is delivered on this path.
func unverifiedRedirectOrigin(token string, conf *buildconf.SKAuthConf) string {
	claims := jwt.MapClaims{}

	if _, _, err := jwt.NewParser().ParseUnverified(token, claims); err != nil {
		return ""
	}

	origin, _ := claims["rdr"].(string)

	if origin == "" || !conf.IsAllowedOrigin(origin) {
		return ""
	}

	return origin
}

// unverifiedReturnTo is unverifiedRedirectOrigin for the OAuth page a sign-in
// started from. It is safe for the same reason: the value is re-validated, and
// oauthReturnTo only ever accepts this host's authorize endpoint.
func (m *skAuthMiddleware) unverifiedReturnTo(token string) string {
	claims := jwt.MapClaims{}

	if _, _, err := jwt.NewParser().ParseUnverified(token, claims); err != nil {
		return ""
	}

	rto, _ := claims["rto"].(string)

	return m.oauthReturnTo(rto)
}

type generateMagicLinkTokenParams struct {
	Env            *buildconf.Env
	RedirectOrigin string
	// CodeChallenge is the PKCE S256 challenge for a custom-scheme redirect. It
	// rides the link so the delivery step can bind the session code to it.
	CodeChallenge string
	// ReturnTo is the OAuth page the sign-in started from, already validated by
	// oauthReturnTo.
	ReturnTo string
}

func generateMagicLinkToken(p generateMagicLinkTokenParams) (string, error) {
	jti, err := utils.SecureRandomToken(16)

	if err != nil {
		return "", err
	}

	claims := jwt.MapClaims{
		"exp": time.Now().Add(15 * time.Minute).Unix(),
		"prv": skauth.ProviderMagicLink,
		"jti": jti,
	}

	if p.RedirectOrigin != "" {
		claims["rdr"] = p.RedirectOrigin
	}

	if p.CodeChallenge != "" {
		claims["cha"] = p.CodeChallenge
	}

	if p.ReturnTo != "" {
		claims["rto"] = p.ReturnTo
	}

	return user.JWT(user.JWTParams{Purpose: user.PurposeSkAuthMagicLink, Claims: claims, Secret: p.Env.AuthConf.Secret})
}

const defaultMagicLinkSubject = "Your magic link"

// errNoMailer reports that a magic link was stored but not sent, because the
// environment has no mailer to send it with.
var errNoMailer = errors.New("no mailer configured")

// defaultMagicLinkBody is used when the provider has no custom body. %[1]s is
// the HTML-escaped site name and %[2]s the HTML-escaped sign-in URL. Layout is
// table-based with inline styles, since that is all email clients render
// reliably.
const defaultMagicLinkBody = `<div style="margin:0;padding:32px 16px;background-color:#f4f4f5;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;">` +
	`<div style="display:none;max-height:0;overflow:hidden;opacity:0;">Your sign-in link for %[1]s. It expires in 15 minutes.</div>` +
	`<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0"><tr><td align="center">` +
	`<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="max-width:480px;background-color:#ffffff;border:1px solid #e4e4e7;border-radius:12px;">` +
	`<tr><td style="padding:40px 40px 32px;">` +
	`<h1 style="margin:0 0 12px;font-size:22px;line-height:30px;font-weight:600;color:#18181b;">Sign in to %[1]s</h1>` +
	`<p style="margin:0 0 28px;font-size:15px;line-height:24px;color:#52525b;">Click the button below to sign in. The link expires in 15 minutes.</p>` +
	`<table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr><td style="border-radius:8px;background-color:#18181b;">` +
	`<a href="%[2]s" style="display:inline-block;padding:12px 28px;font-size:15px;line-height:20px;font-weight:600;color:#ffffff;text-decoration:none;border-radius:8px;">Sign in</a>` +
	`</td></tr></table>` +
	`<p style="margin:32px 0 6px;font-size:13px;line-height:20px;color:#71717a;">Or paste this link into your browser:</p>` +
	`<p style="margin:0;font-size:13px;line-height:20px;word-break:break-all;"><a href="%[2]s" style="color:#52525b;text-decoration:underline;">%[2]s</a></p>` +
	`</td></tr>` +
	`<tr><td style="padding:20px 40px;border-top:1px solid #f4f4f5;font-size:12px;line-height:18px;color:#a1a1aa;">If you did not request this email, you can safely ignore it.</td></tr>` +
	`</table>` +
	`</td></tr></table>` +
	`</div>`

type magicLinkEmail struct {
	Provider *skauth.Provider
	Host     string
	Link     string
}

// subject returns the configured subject, or the default when none is set.
// Newlines are stripped so a stored subject cannot inject SMTP headers.
func (m magicLinkEmail) subject() string {
	return buildconf.SanitizeHeader(utils.GetString(m.Provider.Data.Subject, defaultMagicLinkSubject))
}

// body renders the configured template, or the default one. Only the link is
// escaped: the template itself is HTML authored by the app owner.
func (m magicLinkEmail) body() string {
	link := html.EscapeString(m.Link)

	if tpl := m.Provider.Data.Body; tpl != "" {
		return strings.ReplaceAll(tpl, skauth.MagicLinkPlaceholder, link)
	}

	return fmt.Sprintf(defaultMagicLinkBody, html.EscapeString(m.Host), link)
}

// siteName returns the host the user signs in to: the web origin the link
// sends them back to, or the host the link is served from when there is no
// web origin, such as a same-host or native-app sign-in.
func siteName(redirect, host string) string {
	if u, err := url.Parse(redirect); err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" {
		return u.Host
	}

	return host
}

type sendMagicLinkEmailParams struct {
	Req      *shttp.RequestContext
	Env      *buildconf.Env
	Provider *skauth.Provider
	Email    string
	Token    string

	// Redirect is the origin the link sends the user back to, if any.
	Redirect string
}

// sendMagicLinkEmail stores the email, so the app owner can see it under
// Mailer, then sends it. It returns errNoMailer when there is nothing to send
// it with.
func sendMagicLinkEmail(p sendMagicLinkEmailParams) error {
	u := p.Req.URL()

	msg := magicLinkEmail{
		Provider: p.Provider,
		Host:     siteName(p.Redirect, u.Host),
		Link:     fmt.Sprintf("%s://%s/_stormkit/auth/magic?token=%s", u.Scheme, u.Host, p.Token),
	}

	params := buildconf.SendEmailParams{
		To:      p.Email,
		From:    p.Provider.Data.FromAddress,
		Subject: msg.subject(),
		Body:    msg.body(),
	}

	if err := buildconf.MailerStore().InsertEmail(p.Req.Context(), buildconf.Email{
		EnvID:   p.Env.ID,
		To:      p.Email,
		Subject: params.Subject,
		Body:    params.Body,
	}); err != nil {
		return err
	}

	mailer := p.Env.Mailer()

	if mailer == nil {
		return errNoMailer
	}

	return mailer.Send(params)
}
