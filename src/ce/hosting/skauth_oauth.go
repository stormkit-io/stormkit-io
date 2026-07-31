package hosting

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// oauthProviders is the set of OAuth2 providers served from the hosting layer
// at /_stormkit/auth/<provider>. Email and magic-link have their own paths.
var oauthProviders = map[string]bool{
	skauth.ProviderGoogle: true,
	skauth.ProviderX:      true,
}

// handleOAuthInitiate starts the OAuth2 authorization flow for providerName.
// The environment is derived from the request Host (no envId query param), and
// the post-login redirect target is resolved from the request (see
// resolveRedirect). On success it 302-redirects to the provider's consent
// screen; the provider then calls back the central /v1/auth/callback endpoint.
func (m *skAuthMiddleware) handleOAuthInitiate(providerName string) (*shttp.Response, error) {
	req := m.req.RequestContext
	envID := m.req.Host.Config.EnvID

	if envID == 0 {
		return shttp.NotFound(), nil
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), envID)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get environment by ID %d", envID)), nil
	}

	if env == nil || env.AuthConf == nil || !env.AuthConf.Status {
		return shttp.NotFound(), nil
	}

	prv, err := skauth.NewStore().Provider(req.Context(), envID, providerName)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get provider %s for env %d", providerName, envID)), nil
	}

	if prv == nil || !prv.Status {
		return shttp.NotFound(), nil
	}

	redirect, resp := m.resolveRedirect(env)

	if resp != nil {
		return resp, nil
	}

	// A custom-scheme target delivers the session through a PKCE-bound code, so
	// the challenge is captured here and carried in the state across the provider
	// round trip.
	challenge := req.Query().Get("code_challenge")

	if resp := validateNativeChallenge(redirect, challenge, req.Query().Get("code_challenge_method")); resp != nil {
		return resp, nil
	}

	authURL, err := prv.Client(skauth.ClientParams{
		RedirectURL: m.callbackURL(),
		Secret:      env.AuthConf.Secret,
	}).AuthCodeURL(skauth.AuthCodeURLParams{
		EnvID:         envID,
		ProviderName:  providerName,
		Referrer:      redirect,
		Secret:        env.AuthConf.Secret,
		CodeChallenge: challenge,
	})

	if err != nil {
		return shttp.Error(err, "failed to build authorization URL"), nil
	}

	return &shttp.Response{
		Status:   http.StatusFound,
		Redirect: &authURL,
	}, nil
}

// callbackURL is the absolute OAuth2 redirect URI for this host. It must be
// identical in the authorization request and the token exchange, so it is
// derived from the request Host in both handleOAuthInitiate and
// handleOAuthCallback (which land on the same domain).
func (m *skAuthMiddleware) callbackURL() string {
	return "https://" + m.req.Host.Name + skauth.CallbackPath
}

// handleOAuthCallback completes the OAuth2 flow: it exchanges the authorization
// code for a token, upserts the auth user, mints a session token, and hands the
// browser back to the origin that started the flow via a one-time code (the
// session token is stored in Redis and injected into localStorage on landing).
func (m *skAuthMiddleware) handleOAuthCallback() (*shttp.Response, error) {
	req := m.req.RequestContext

	// The app's own origin is always a safe redirect target. It is used for any
	// failure that happens before the state referrer is validated, and as the
	// fallback whenever that referrer is not (or no longer) allow-listed.
	ownOrigin := "https://" + m.req.Host.Name

	envID := m.req.Host.Config.EnvID

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), envID)

	if err != nil {
		slog.Errorf("oauth callback: failed to get environment %d: %s", envID, err.Error())
		return m.loginErrorRedirect(ownOrigin, "Sign-in is temporarily unavailable. Please try again."), nil
	}

	if !env.AuthReady() {
		return m.loginErrorRedirect(ownOrigin, "Sign-in is not available right now."), nil
	}

	// State is signed with the environment secret, so a token minted for another
	// tenant, or one past its short expiry, fails to parse and is rejected.
	claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: req.FormValue("state"), Secret: env.AuthConf.Secret})

	provider, ok := claims["prv"].(string)
	refer, refOK := claims["ref"].(string)

	if !ok || !refOK || !oauthProviders[provider] {
		return shttp.BadRequest(map[string]any{"errors": []string{"invalid state parameter"}}), nil
	}

	// The session cookie is planted on, and the browser redirected to, this
	// origin. Re-validate it against the allow-list rather than trusting the
	// referrer carried in the state, so a crafted state cannot turn the callback
	// into an open redirect that also delivers a valid session.
	referOrigin := m.validateReferOrigin(env, refer, ownOrigin)

	// The provider round-trips an `error` param (e.g. the user denied access)
	// instead of an authorization code. Bounce back without attempting an
	// exchange that would fail anyway.
	if req.FormValue("error") != "" {
		return m.loginErrorRedirect(referOrigin, "Sign-in was cancelled."), nil
	}

	prv, err := skauth.NewStore().Provider(req.Context(), env.ID, provider)

	if err != nil {
		slog.Errorf("oauth callback: failed to get provider %s for env %d: %s", provider, env.ID, err.Error())
		return m.loginErrorRedirect(referOrigin, "Sign-in is temporarily unavailable. Please try again."), nil
	}

	if prv == nil || !prv.Status {
		return m.loginErrorRedirect(referOrigin, "This sign-in method is not available."), nil
	}

	client := prv.Client(skauth.ClientParams{
		RedirectURL: m.callbackURL(),
		Secret:      env.AuthConf.Secret,
	})

	if client == nil {
		return m.loginErrorRedirect(referOrigin, "This sign-in method is not available."), nil
	}

	token, err := client.Exchange(req.Context(), req)

	if err != nil {
		slog.Errorf("oauth callback: failed to exchange authorization code: %s", err.Error())
		return m.loginErrorRedirect(referOrigin, "We couldn't complete sign-in. Please try again."), nil
	}

	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeAppUser)

	if err != nil {
		slog.Errorf("oauth callback: failed to get schema store: %s", err.Error())
		return m.loginErrorRedirect(referOrigin, "Sign-in is temporarily unavailable. Please try again."), nil
	}

	info, err := client.UserInfo(req.Context(), token)

	if err != nil {
		slog.Errorf("oauth callback: failed to get user info: %s", err.Error())
		return m.loginErrorRedirect(referOrigin, "We couldn't read your account details. Please try again."), nil
	}

	if info == nil {
		return m.loginErrorRedirect(referOrigin, "We couldn't read your account details. Please try again."), nil
	}

	email := normalizeEmail(info.Email)

	// Accounts are linked across providers by email, so a provider that hands
	// back an unverified (attacker-controllable) address must never be allowed
	// to attach to — and sign in as — an existing user.
	if email == "" || !info.EmailVerified {
		return m.loginErrorRedirect(referOrigin, "Your email address must be verified with the provider to sign in."), nil
	}

	oauth := skauth.OAuth{
		AccountID:    info.AccountID,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       utils.UnixFrom(token.Expiry),
		ProviderName: provider,
	}

	usr := skauth.User{
		Email:     email,
		Avatar:    info.Avatar,
		FirstName: info.FirstName,
		LastName:  info.LastName,
		Metadata:  info.UserMetadata,
	}

	if err := store.UpsertAuthUser(req.Context(), &oauth, &usr); err != nil {
		slog.Errorf("oauth callback: failed to upsert auth user: %s", err.Error())
		return m.loginErrorRedirect(referOrigin, "We couldn't complete sign-in. Please try again."), nil
	}

	// Metadata lives on the user row and is merged on every login, so a provider
	// that supplies no handle can't clobber values another provider stored on the
	// shared, email-linked row. Non-fatal: it must not block sign-in if the write
	// fails, and if the metadata column isn't reconciled yet (see EnsureAuthSchemas
	// job) this simply no-ops until it is.
	if err := store.UpdateAuthUserMetadata(req.Context(), usr.UUID, usr.Metadata); err != nil {
		slog.Errorf("oauth callback: failed to update user metadata: %s", err.Error())
	}

	if err := store.UpdateLastLogin(req.Context(), usr.ID); err != nil {
		slog.Errorf("oauth callback: failed to update last login: %s", err.Error())
	}

	sessionToken, err := user.JWT(jwt.MapClaims{
		"uid": usr.UUID,
		"eml": utils.EncryptToString(usr.Email, emlKey(env.AuthConf.Secret)),
		"eid": fmt.Sprintf("%d", env.ID),
		"prv": provider,
	}, env.AuthConf.Secret)

	if err != nil {
		slog.Errorf("oauth callback: failed to generate session token: %s", err.Error())
		return m.loginErrorRedirect(referOrigin, "We couldn't complete sign-in. Please try again."), nil
	}

	successURL := m.req.Host.Config.SKAuth.SuccessURL

	// The challenge is only honoured together with the referrer it was minted
	// alongside; validateReferOrigin may have fallen back to this host, in which
	// case delivery is a plain first-party cookie and the challenge is moot.
	challenge, _ := claims["cha"].(string)

	// Set a first-party cookie on this auth host (also the OAuth AS) and send the
	// user straight to the initiating origin — no dependency on that origin
	// carrying its own auth config. A native app initiating with a custom-scheme
	// origin gets a PKCE-bound code on the redirect instead; see deliverSession.
	return m.deliverSession(deliverSessionParams{
		Token:     sessionToken,
		Origin:    referOrigin,
		Path:      successURL,
		Challenge: challenge,
	}), nil
}

// resolveRedirect determines and validates the post-login redirect origin.
// Precedence: ?redirect= query param → Origin header → Referer header. When the
// environment configures an AllowedOrigins allow-list, a cross-origin target
// must be on it (otherwise 403). When no target is supplied — or none is
// allow-listed — the flow falls back to the app's own origin (the single-host
// case). The returned value is always scheme + host with no path.
func (m *skAuthMiddleware) resolveRedirect(env *buildconf.Env) (string, *shttp.Response) {
	req := m.req.RequestContext

	candidate := utils.GetString(
		req.Query().Get("redirect"),
		m.req.Header.Get("Origin"),
		m.req.Header.Get("Referer"),
	)

	// The app's own origin — the fallback whenever no allow-listed cross-origin
	// target applies. Hosting always serves over HTTPS in production.
	own := "https://" + m.req.Host.Name

	if candidate == "" {
		return own, nil
	}

	candidate, ok := originOf(candidate)

	if !ok {
		return "", shttp.BadRequest(map[string]any{
			"errors": []string{"redirect is not a valid absolute URL"},
		})
	}

	if len(env.AuthConf.AllowedOrigins) > 0 {
		if !env.AuthConf.IsAllowedOrigin(candidate) {
			return "", &shttp.Response{
				Status: http.StatusForbidden,
				Data:   map[string]any{"errors": []string{"redirect origin is not allowed"}},
			}
		}

		return candidate, nil
	}

	// No allow-list configured: ignore the cross-origin target and stay
	// single-host so the session lands back on this domain.
	return own, nil
}

// validateReferOrigin resolves the origin the session is delivered to on
// callback. It mirrors resolveRedirect's allow-list logic so the callback
// cannot be steered to an origin the initiate step would have rejected: the
// referrer is honoured only when it is on the environment's allow-list,
// otherwise the flow stays first-party on the app's own origin.
func (m *skAuthMiddleware) validateReferOrigin(env *buildconf.Env, refer, own string) string {
	origin, ok := originOf(refer)

	if ok && len(env.AuthConf.AllowedOrigins) > 0 && env.AuthConf.IsAllowedOrigin(origin) {
		return origin
	}

	return own
}

// originOf reduces an absolute URL to its scheme://host origin (scheme
// defaulting to https), with no path. ok is false when candidate is not a valid
// absolute URL. It is the single place both the initiate and callback paths
// derive an origin from, so their allow-list matching cannot drift on how a URL
// is canonicalized.
func originOf(candidate string) (origin string, ok bool) {
	parsed, err := url.ParseRequestURI(candidate)

	if err != nil {
		return "", false
	}

	return utils.GetString(parsed.Scheme, "https") + "://" + parsed.Host, true
}

// isOAuthProviderPath reports whether path is /_stormkit/auth/<oauth-provider>
// and returns the provider name when it is.
func isOAuthProviderPath(path string) (string, bool) {
	provider, ok := strings.CutPrefix(path, "/_stormkit/auth/")

	if !ok || !oauthProviders[provider] {
		return "", false
	}

	return provider, true
}
