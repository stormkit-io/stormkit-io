package hosting

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/html"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type skAuthMiddleware struct {
	req *RequestContext
}

// sessionBearer returns the SkAuth session token for the request, consulting
// both credentials: the Authorization bearer (localStorage mode and API
// clients) and the session cookie (cookie mode, where a top-level navigation or
// a credentials:'include' XHR carries no Authorization header).
//
// In cookie mode the cookie is preferred so a stale Authorization bearer — e.g.
// a token left in localStorage from before the switch to cookie mode, which the
// old SPA still attaches to every XHR — can't mask the live cookie session. API
// and MCP clients send no cookie, so the Authorization bearer remains the
// fallback. In localStorage mode there is no session cookie, so only the bearer
// applies.
func sessionBearer(req *RequestContext) string {
	if req.Host.Config.SKAuth.SessionInCookie() {
		if cookie, err := req.Cookie(buildconf.SessionCookieName); err == nil && cookie != nil && cookie.Value != "" {
			return cookie.Value
		}
	}

	return user.ParseBearer(req.Header.Get("Authorization"))
}

// sessionCookieFor builds the SkAuth session cookie carrying token. It is
// Secure + HttpOnly (the edge reads it server-side; the SPA uses /me), and
// SameSite=Lax so it still rides the top-level navigation into the OAuth
// /authorize endpoint while resisting CSRF. A configured CookieDomain shares it
// across subdomains (login origin vs. authorization-server origin).
func (m *skAuthMiddleware) sessionCookieFor(token string) http.Cookie {
	conf := m.req.Host.Config.SKAuth

	cookie := http.Cookie{
		Name:     buildconf.SessionCookieName,
		Value:    token,
		Path:     "/",
		Domain:   conf.CookieDomain,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	if conf.TTL > 0 {
		cookie.MaxAge = conf.TTL * 60
	}

	return cookie
}

// deliverSession hands the freshly minted session token to the browser and
// sends it on to redirectURL. In cookie mode it sets the shared session cookie
// and 302s; in localStorage mode it renders the landing page whose script
// stashes the token before redirecting. Callers compute redirectURL as either a
// relative path or an absolute URL on the destination origin.
func (m *skAuthMiddleware) deliverSession(token, redirectURL string) *shttp.Response {
	if m.req.Host.Config.SKAuth.SessionInCookie() {
		return &shttp.Response{
			Status:   http.StatusFound,
			Redirect: &redirectURL,
			Cookies:  []http.Cookie{m.sessionCookieFor(token)},
			Headers: shttp.HeadersFromMap(map[string]string{
				"Cache-Control":   "no-store",
				"Referrer-Policy": "no-referrer",
			}),
		}
	}

	head := fmt.Sprintf(
		`<script>localStorage.setItem('skauth', JSON.stringify(%q));window.location.href=%q;</script>`,
		token,
		redirectURL,
	)

	return m.renderVerifyPage(http.StatusOK, head, "")
}

func (m *skAuthMiddleware) handleRegisterLogin(path string) (*shttp.Response, error) {
	if m.req.Method != http.MethodPost {
		return &shttp.Response{
			Status: http.StatusMethodNotAllowed,
			Data:   map[string]any{"errors": []string{"method not allowed"}},
		}, nil
	}

	if path == "/_stormkit/auth/login" {
		return m.login(), nil
	}

	return m.register(), nil
}

func (m *skAuthMiddleware) handleVerify() (*shttp.Response, error) {
	if m.req.Method != http.MethodGet {
		return &shttp.Response{
			Status: http.StatusMethodNotAllowed,
			Data:   map[string]any{"errors": []string{"method not allowed"}},
		}, nil
	}

	resp := m.verifyEmail()

	if resp.Status != 0 && resp.Status != http.StatusOK {
		errMsg := "verification failed"

		if d, ok := resp.Data.(map[string]any); ok {
			if errs, ok := d["errors"].([]string); ok && len(errs) > 0 {
				errMsg = errs[0]
			}
		}

		return m.renderVerifyPage(resp.Status, "", errMsg), nil
	}

	data, ok := resp.Data.(map[string]any)

	if !ok {
		return m.renderVerifyPage(http.StatusInternalServerError, "", "verification failed"), nil
	}

	token, _ := data["token"].(string)

	return m.deliverSession(token, m.req.Host.Config.SKAuth.SuccessURL+"?verified=true"), nil
}

func (m *skAuthMiddleware) handleMagicLinkRequest() (*shttp.Response, error) {
	resp := m.magicLinkRequest()

	if resp.Status != 0 && resp.Status != http.StatusOK {
		return resp, nil
	}

	return &shttp.Response{Status: http.StatusCreated}, nil
}

func (m *skAuthMiddleware) handleMagicLinkVerify() (*shttp.Response, error) {
	resp := m.magicLinkVerify()

	if resp.Status != 0 && resp.Status != http.StatusOK {
		errMsg := "magic link verification failed"

		// Redirect base: the validated origin carried in the token when we have
		// it (cross-origin), otherwise this host's own origin — which, for magic
		// links, is the domain the email link was issued from. The underlying
		// cause was already logged when magicLinkVerify built the error response.
		origin := "https://" + m.req.Host.Name

		if d, ok := resp.Data.(map[string]any); ok {
			if errs, ok := d["errors"].([]string); ok && len(errs) > 0 {
				errMsg = errs[0]
			}

			if r, ok := d["redirect"].(string); ok && r != "" {
				origin = r
			}
		}

		// Token-level failures (missing/invalid/expired/consumed token) bounce the
		// user back to the app with a friendly login_error. Config (404) and
		// internal (5xx) responses keep their original page — they mean the host
		// isn't set up for auth or hit a fault, not a user sign-in failure.
		if resp.Status == http.StatusBadRequest {
			return m.loginErrorRedirect(origin, errMsg), nil
		}

		return m.renderVerifyPage(resp.Status, "", errMsg), nil
	}

	data, ok := resp.Data.(map[string]any)

	if !ok {
		return m.renderVerifyPage(http.StatusInternalServerError, "", "magic link verification failed"), nil
	}

	successURL := m.req.Host.Config.SKAuth.SuccessURL
	token, _ := data["token"].(string)

	// Cross-origin: the POST came from a whitelisted frontend. In cookie mode the
	// session rides a first-party cookie set on this auth host (see
	// crossOriginCookieResponse); otherwise fall back to the localStorage bounce.
	if redirect, _ := data["redirect"].(string); redirect != "" {
		if res := m.crossOriginCookieResponse(redirect, token); res != nil {
			return res, nil
		}

		landing, err := stashSessionToken(m.req.Context(), redirect, successURL, token)

		if err != nil {
			return m.renderVerifyPage(http.StatusInternalServerError, "", "magic link verification failed"), nil
		}

		head := fmt.Sprintf(`<script>window.location.href=%q;</script>`, landing)

		return m.renderVerifyPage(http.StatusOK, head, ""), nil
	}

	return m.deliverSession(token, successURL+"?verified=true"), nil
}

// crossOriginCookieResponse returns the cookie-mode delivery for a login that
// lands on a different origin (magic-link, OAuth-provider): the session cookie
// is set first-party on this auth host — which is also the OAuth authorization
// server — and the browser is sent straight to origin+successURL. This removes
// the localStorage bounce and, crucially in a decoupled setup, stops the landing
// from depending on the frontend host carrying its own cookie-mode config. It
// returns nil when the environment is not in cookie mode, so the caller keeps
// its existing localStorage delivery. origin is scheme+host (no trailing slash);
// SuccessURL is validated at config time to start with "/".
func (m *skAuthMiddleware) crossOriginCookieResponse(origin, token string) *shttp.Response {
	if !m.req.Host.Config.SKAuth.SessionInCookie() {
		return nil
	}

	return m.deliverSession(token, origin+m.req.Host.Config.SKAuth.SuccessURL)
}

// finalizeSessionResponse adapts a successful JSON auth response (login,
// register, refresh) to the environment's session-storage mode. In cookie mode
// it sets the session cookie and removes the token from the body, so the browser
// holds exactly one credential — the cookie — and the app can't drift by also
// stashing the token in localStorage. In localStorage mode the response is
// returned unchanged, with the token in the body for the app to store.
//
// In cookie mode a decoupled frontend must send the auth XHR with credentials,
// and the app's CORS config must allow them, for the browser to store the cookie.
func (m *skAuthMiddleware) finalizeSessionResponse(res *shttp.Response, token string) *shttp.Response {
	if res == nil || res.Status >= http.StatusBadRequest || !m.req.Host.Config.SKAuth.SessionInCookie() {
		return res
	}

	// A session cookie must never be planted on a cross-site request: an attacker
	// who auto-submits a cross-site form login (text/plain body, no CORS
	// preflight) would otherwise fixate the victim's browser with the attacker's
	// session cookie. Browsers always send Origin on POST and cookie mode only
	// benefits browsers, so a missing or non-allow-listed Origin is treated as
	// cross-site and the login is refused before the cookie is set.
	if !m.cookieOriginAllowed() {
		return &shttp.Response{
			Status: http.StatusForbidden,
			Data:   map[string]any{"errors": []string{"cross-origin session request rejected"}},
		}
	}

	res.Cookies = append(res.Cookies, m.sessionCookieFor(token))

	if data, ok := res.Data.(map[string]any); ok {
		delete(data, "token")
	}

	return res
}

// cookieOriginAllowed reports whether the request that is about to receive a
// session cookie originates from a trusted browser context. Same-host requests
// and configured AllowedOrigins pass; a missing or foreign Origin is rejected.
// This gates the cookie-setting POST endpoints (login, register, refresh)
// against login CSRF / session fixation.
func (m *skAuthMiddleware) cookieOriginAllowed() bool {
	origin := m.req.Header.Get("Origin")

	if origin == "" {
		return false
	}

	if origin == "https://"+m.req.Host.Name {
		return true
	}

	return m.req.Host.Config.SKAuth.IsAllowedOrigin(origin)
}

// sessionCode is the payload stored in Redis under the one-time login code. The
// auth host mints the session token and computes the post-login redirect, then
// stashes both so the landing page can inject the token and redirect — without
// the landing host needing its own auth config.
type sessionCode struct {
	Token    string `json:"token"`
	Redirect string `json:"redirect"`
}

// magicLinkCodeStore holds the one-time login codes for the magic-link landing.
// The empty prefix keeps the bare-code keyspace the landing URL embeds; OAuth
// codes live under their own prefix so the two never collide.
var magicLinkCodeStore = oneTimeCode{prefix: "", ttl: 2 * time.Minute}

// stashSessionToken stores the session token and its post-login redirect under a
// fresh one-time code and returns the landing URL on the destination origin. The
// browser is sent there; codeLanding reads the payload back and injects the token
// into localStorage on that origin. origin is scheme+host (no trailing slash);
// successURL is validated at config time to start with '/'.
func stashSessionToken(ctx context.Context, origin, successURL, token string) (string, error) {
	code, err := magicLinkCodeStore.issue(ctx, sessionCode{Token: token, Redirect: origin + successURL})

	if err != nil {
		return "", err
	}

	return origin + "/_stormkit/auth?code=" + code, nil
}

// loginErrorRedirect bounces a failed sign-in back to the app's redirect page
// (origin + SuccessURL) with a user-friendly message in the login_error query
// param, instead of rendering a Stormkit error page. The underlying cause should
// be logged by the caller. Used by both the OAuth and magic-link flows.
func (m *skAuthMiddleware) loginErrorRedirect(origin, message string) *shttp.Response {
	target := fmt.Sprintf("%s%s?login_error=%s",
		origin,
		m.req.Host.Config.SKAuth.SuccessURL,
		url.QueryEscape(message),
	)

	return &shttp.Response{
		Status:   http.StatusFound,
		Redirect: &target,
	}
}

// codeLanding serves the one-time-code login landing. It is host-config
// independent: the token and redirect were stashed in Redis by the auth host, so
// it works even on a frontend deployment that has no auth config of its own.
// Returns nil when the code is absent or unknown, so the request falls through to
// normal serving (the SPA) instead of rendering an error.
func (m *skAuthMiddleware) codeLanding() *shttp.Response {
	code := m.req.Query().Get("code")

	if code == "" {
		return nil
	}

	var sc sessionCode

	if err := magicLinkCodeStore.redeem(m.req.Context(), code, &sc); err != nil {
		return nil
	}

	// This landing runs on the destination origin, so a cookie set here (with
	// the configured Domain) is visible to the app and its authorization
	// server — exactly the cross-subdomain boundary the bounce exists to cross.
	return m.deliverSession(sc.Token, sc.Redirect)
}

// handleCallback renders the terminal states of the one-time-code landing on an
// auth-enabled host. The success path (a valid code) is handled earlier by
// codeLanding, so by the time we get here the code is missing or unknown.
func (m *skAuthMiddleware) handleCallback() (*shttp.Response, error) {
	status := http.StatusOK
	content := "invalid session"

	if m.req.Query().Get("code") == "" {
		status = http.StatusBadRequest
		content = "code is missing"
	}

	return &shttp.Response{
		Status: status,
		Headers: shttp.HeadersFromMap(map[string]string{
			"Content-Type": "text/html",
		}),
		Data: html.MustRender(html.RenderArgs{
			PageTitle:   "Stormkit - Auth",
			PageContent: content,
		}),
	}, nil
}

func (m *skAuthMiddleware) renderVerifyPage(status int, head, content string) *shttp.Response {
	return &shttp.Response{
		Status: status,
		Headers: shttp.HeadersFromMap(map[string]string{
			"Content-Type":    "text/html",
			"Referrer-Policy": "no-referrer",
			"Cache-Control":   "no-store",
		}),
		Data: html.MustRender(html.RenderArgs{
			PageTitle:   "Stormkit - Auth",
			PageHead:    head,
			PageContent: content,
		}),
	}
}

// WithSKAuth carries the cross-cutting auth concerns that must run for every app
// request: the config-independent one-time-code login landing and the
// verified-bearer -> identity header injection. The discrete /_stormkit/auth/*
// endpoints (register, login, verify, magic, refresh, me, callback, provider)
// are served by dedicated routes — see registerReservedRoutes.
func WithSKAuth(req *RequestContext) (*shttp.Response, error) {
	path := req.URL().Path

	// The one-time-code login landing works on any Stormkit host: the token and
	// its redirect live in Redis (stashed by the auth host), so the destination
	// frontend doesn't need its own auth config. This must run before the
	// SKAuth-nil gate below. The exact-match compare keeps normal traffic free —
	// the query is parsed only on /_stormkit/auth, and Redis is touched only when
	// a code is actually present. codeLanding returns nil for an absent/unknown
	// code so we fall through to normal serving (the SPA). Skip OPTIONS so a CORS
	// preflight can't consume the one-time code (real landings are GET
	// navigations and are never preflighted).
	if path == "/_stormkit/auth" && req.Method != http.MethodOptions && req.Query().Get("code") != "" {
		if resp := (&skAuthMiddleware{req: req}).codeLanding(); resp != nil {
			return resp, nil
		}
	}

	if req.Host.Config.SKAuth == nil {
		return nil, nil
	}

	injectUserHeaders(req)

	// Only the bare landing remains here; its terminal states (missing/unknown
	// code) render the auth page. Everything under /_stormkit/auth/ is routed.
	if path != "/_stormkit/auth" {
		return nil, nil
	}

	if req.Method == http.MethodOptions {
		return &shttp.Response{Status: http.StatusNoContent}, nil
	}

	return (&skAuthMiddleware{req: req}).handleCallback()
}

// injectUserHeaders strips any client-supplied identity headers and, when a
// valid skauth bearer accompanies the request, re-injects the authenticated
// X-User-Id / X-User-Email for downstream handlers and the customer app.
func injectUserHeaders(req *RequestContext) {
	// Strip the headers to prevent clients from spoofing them.
	req.Header.Del("X-User-Id")
	req.Header.Del("X-User-Email")

	bearer := sessionBearer(req)

	if bearer == "" {
		return
	}

	secret := req.Host.Config.SKAuth.Secret

	claims := user.ParseJWT(&user.ParseJWTArgs{
		Bearer:  bearer,
		Secret:  secret,
		MaxMins: req.Host.Config.SKAuth.TTL,
	})

	if claims == nil {
		return
	}

	if userID, ok := claims["uid"].(string); ok && userID != "" {
		req.Header.Set("X-User-Id", userID)
	}

	if encEmail, ok := claims["eml"].(string); ok && encEmail != "" {
		if email := utils.DecryptToString(encEmail, emlKey(secret)); email != "" {
			req.Header.Set("X-User-Email", email)
		}
	}
}

// handleMagic dispatches the magic-link endpoint: a token query parameter means
// the user is following a link (verify), otherwise it's a request to send one.
func (m *skAuthMiddleware) handleMagic() (*shttp.Response, error) {
	if m.req.Query().Get("token") != "" {
		return m.handleMagicLinkVerify()
	}

	return m.handleMagicLinkRequest()
}

// handleRefresh trades a currently-valid skauth bearer for a freshly signed
// one carrying the same uid claim. Expired bearers are rejected — the user
// must re-run the magic-link flow. Clients call this proactively (e.g. on
// app open and on a periodic timer) so the token never reaches its TTL while
// the user is active.
func (m *skAuthMiddleware) handleRefresh() (*shttp.Response, error) {
	if m.req.Method != http.MethodPost {
		return &shttp.Response{
			Status: http.StatusMethodNotAllowed,
			Data:   map[string]any{"errors": []string{"method not allowed"}},
		}, nil
	}

	bearer := sessionBearer(m.req)

	if bearer == "" {
		return &shttp.Response{
			Status: http.StatusUnauthorized,
			Data:   map[string]any{"errors": []string{"missing bearer token"}},
		}, nil
	}

	claims := user.ParseJWT(&user.ParseJWTArgs{
		Bearer:  bearer,
		Secret:  m.req.Host.Config.SKAuth.Secret,
		MaxMins: m.req.Host.Config.SKAuth.TTL,
	})

	if claims == nil {
		return &shttp.Response{
			Status: http.StatusUnauthorized,
			Data:   map[string]any{"errors": []string{"invalid or expired token"}},
		}, nil
	}

	uid, _ := claims["uid"].(string)

	if uid == "" {
		return &shttp.Response{
			Status: http.StatusUnauthorized,
			Data:   map[string]any{"errors": []string{"invalid token claims"}},
		}, nil
	}

	newClaims := jwt.MapClaims{"uid": uid, "eml": claims["eml"]}

	token, err := user.JWT(newClaims, m.req.Host.Config.SKAuth.Secret)

	if err != nil {
		return &shttp.Response{
			Status: http.StatusInternalServerError,
			Data:   map[string]any{"errors": []string{"failed to sign token"}},
		}, nil
	}

	// finalizeSessionResponse rotates the cookie and drops the token from the body
	// in cookie mode; in localStorage mode it returns the token for the SPA to
	// persist.
	return m.finalizeSessionResponse(&shttp.Response{
		Status: http.StatusOK,
		Data:   map[string]any{"token": token},
	}, token), nil
}

// handleMe returns the full profile for the user identified by the bearer token
// — the OIDC-style userinfo endpoint. Per-request headers stay limited to
// X-User-Id / X-User-Email; richer, sync-once fields (name, avatar, provider
// username and profile link) are served here instead of bloating every request.
func (m *skAuthMiddleware) handleMe() (*shttp.Response, error) {
	if m.req.Method != http.MethodGet {
		return &shttp.Response{
			Status: http.StatusMethodNotAllowed,
			Data:   map[string]any{"errors": []string{"method not allowed"}},
		}, nil
	}

	// X-User-Id is injected by WithSKAuth from the verified bearer before
	// dispatch, so a value here is already authenticated. Empty means no valid
	// token accompanied the request. Identity comes solely from the token — the
	// endpoint can only ever return the caller's own record.
	uid := m.req.Header.Get("X-User-Id")

	if uid == "" {
		return &shttp.Response{
			Status: http.StatusUnauthorized,
			Data:   map[string]any{"errors": []string{"missing or invalid token"}},
		}, nil
	}

	envID := m.req.Host.Config.EnvID

	if envID == 0 {
		return shttp.NotFound(), nil
	}

	env, err := buildconf.NewStore().EnvironmentByID(m.req.Context(), envID)

	if err != nil {
		return shttp.Error(err, "handleMe: failed to get environment"), nil
	}

	if !env.AuthReady() {
		return shttp.NotFound(), nil
	}

	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeAppUser)

	if err != nil {
		return shttp.Error(err, "handleMe: failed to get schema store"), nil
	}

	usr, err := store.AuthUserByUUID(m.req.Context(), uid)

	if err != nil {
		return shttp.Error(err, "handleMe: failed to get auth user"), nil
	}

	if usr == nil {
		return shttp.NotFound(), nil
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   usr.JSON(),
	}, nil
}
