package hosting

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

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
// both credentials: the session cookie (browsers — a top-level navigation or a
// credentials:'include' XHR carries no Authorization header) and the
// Authorization bearer (native/mobile and MCP clients, which send no cookie).
//
// The cookie is preferred so a stale Authorization bearer — e.g. a token an old
// SPA still attaches to every XHR — can't mask the live cookie session.
func sessionBearer(req *RequestContext) string {
	if cookie, err := req.Cookie(buildconf.SessionCookieName); err == nil && cookie != nil && cookie.Value != "" {
		return cookie.Value
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

// expiredSessionCookie is the deletion counterpart of sessionCookieFor: the same
// Name/Path/Domain (which the browser matches on) with MaxAge<0, so a Set-Cookie
// carrying it clears the session cookie.
func (m *skAuthMiddleware) expiredSessionCookie() http.Cookie {
	return http.Cookie{
		Name:     buildconf.SessionCookieName,
		Value:    "",
		Path:     "/",
		Domain:   m.req.Host.Config.SKAuth.CookieDomain,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
}

// handleLogout clears the session cookie. The session is an HttpOnly cookie the
// app's JS cannot clear, so sign-out must expire it server side. Registered
// POST-only (see registerReservedRoutes), so a cross-site GET can't force it.
func (m *skAuthMiddleware) handleLogout() (*shttp.Response, error) {
	return &shttp.Response{
		Status:  http.StatusOK,
		Cookies: []http.Cookie{m.expiredSessionCookie()},
	}, nil
}

// deliverSessionParams describes where a freshly minted session token is headed.
// Origin is the already allow-list-validated destination origin (scheme + host,
// no path) and is empty for a same-host landing. Path is the relative landing
// path — the configured SuccessURL, sometimes carrying a query — appended to
// Origin for browser targets.
type deliverSessionParams struct {
	Token  string
	Origin string
	Path   string
}

// deliverSession hands the session token to the client and 302s onward. Used by
// the login landings (email verify, magic link, OAuth-provider callback).
//
// Browsers receive it as the HttpOnly session cookie — first-party to this auth
// host, so it is also readable by the OAuth /authorize navigation — and land on
// Origin+Path. A native app, identified by a custom-scheme Origin such as
// triplan://auth, has no usable cookie jar: the system auth session
// (ASWebAuthenticationSession / Custom Tabs) discards Set-Cookie. For those the
// token rides the redirect URL instead and no cookie is set. The scheme is
// claimed by the owning app, so the OS hands the URL only to it, and Origin has
// already been matched against the configured allow-list by the caller.
func (m *skAuthMiddleware) deliverSession(p deliverSessionParams) *shttp.Response {
	headers := shttp.HeadersFromMap(map[string]string{
		"Cache-Control":   "no-store",
		"Referrer-Policy": "no-referrer",
	})

	if isNativeSchemeOrigin(p.Origin) {
		target := p.Origin + "?token=" + url.QueryEscape(p.Token)

		return &shttp.Response{
			Status:   http.StatusFound,
			Redirect: &target,
			Headers:  headers,
		}
	}

	target := p.Origin + p.Path

	return &shttp.Response{
		Status:   http.StatusFound,
		Redirect: &target,
		Cookies:  []http.Cookie{m.sessionCookieFor(p.Token)},
		Headers:  headers,
	}
}

// isNativeSchemeOrigin reports whether origin uses a custom (non-http(s)) scheme
// such as triplan://auth — the redirect target a native app registers in the
// environment's allowed origins. It only classifies the scheme; the caller is
// responsible for having validated origin against the allow-list first.
func isNativeSchemeOrigin(origin string) bool {
	scheme, _, ok := strings.Cut(origin, "://")

	if !ok || scheme == "" {
		return false
	}

	scheme = strings.ToLower(scheme)

	return scheme != "http" && scheme != "https"
}

func (m *skAuthMiddleware) handleRegisterLogin(path string) (*shttp.Response, error) {
	if path == "/_stormkit/auth/login" {
		return m.login(), nil
	}

	return m.register(), nil
}

func (m *skAuthMiddleware) handleVerify() (*shttp.Response, error) {
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

	return m.deliverSession(deliverSessionParams{
		Token: token,
		Path:  m.req.Host.Config.SKAuth.SuccessURL + "?verified=true",
	}), nil
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

	// Cross-origin: the request came from a whitelisted frontend, whose origin the
	// magic-link token carried in its `rdr` claim and magicLinkVerify re-validated
	// against the allow-list. The session cookie is first-party to this auth host
	// (also the OAuth AS), so it is set here and the browser is sent straight to
	// the destination origin — no dependency on the frontend host carrying its own
	// auth config. A native custom-scheme origin gets the token in the URL instead.
	if redirect, _ := data["redirect"].(string); redirect != "" {
		return m.deliverSession(deliverSessionParams{
			Token:  token,
			Origin: redirect,
			Path:   successURL,
		}), nil
	}

	return m.deliverSession(deliverSessionParams{
		Token: token,
		Path:  successURL + "?verified=true",
	}), nil
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
	if res == nil || res.Status >= http.StatusBadRequest || !m.wantsCookieDelivery() {
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

// sessionDeliveryHeader lets a native/mobile client opt into bearer delivery on
// a cookie-mode environment: it has no cookie jar, so it asks for the token in
// the response body and sends it back as an Authorization bearer. Browsers omit
// the header and get the cookie.
const (
	sessionDeliveryHeader = "X-Session-Delivery"
	sessionDeliveryBearer = "bearer"
)

// wantsCookieDelivery reports whether the session should be delivered as a
// cookie (the browser default) rather than a bearer token in the response body.
// Bearer delivery — token in the body, no cookie, and no cookie-CSRF Origin gate
// — applies when either:
//
//   - the client opts in via the X-Session-Delivery header (native/mobile at
//     login, where there is no session credential yet to infer from); or
//   - the request authenticated with an Authorization bearer and carries no
//     session cookie (a native client rotating its token at /refresh) — so it
//     "just works" without the header. A browser always presents the cookie, so
//     the cookie wins even if a stale Authorization header rides along.
func (m *skAuthMiddleware) wantsCookieDelivery() bool {
	if strings.EqualFold(m.req.Header.Get(sessionDeliveryHeader), sessionDeliveryBearer) {
		return false
	}

	if cookie, err := m.req.Cookie(buildconf.SessionCookieName); (err != nil || cookie.Value == "") &&
		user.ParseBearer(m.req.Header.Get("Authorization")) != "" {
		return false
	}

	return true
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

// loginErrorRedirect bounces a failed sign-in back to the app's redirect page
// (origin + SuccessURL) with a user-friendly message in the login_error query
// param, instead of rendering a Stormkit error page. The underlying cause should
// be logged by the caller. Used by both the OAuth and magic-link flows.
//
// A native custom-scheme origin has no SuccessURL page to land on — it is the
// deep link itself — so the message is hung off the bare origin, mirroring how
// deliverSession returns the token there.
func (m *skAuthMiddleware) loginErrorRedirect(origin, message string) *shttp.Response {
	path := m.req.Host.Config.SKAuth.SuccessURL

	if isNativeSchemeOrigin(origin) {
		path = ""
	}

	target := fmt.Sprintf("%s%s?login_error=%s",
		origin,
		path,
		url.QueryEscape(message),
	)

	return &shttp.Response{
		Status:   http.StatusFound,
		Redirect: &target,
	}
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

// WithSKAuth injects the verified-bearer identity headers (X-User-Id /
// X-User-Email) for every app request. The discrete /_stormkit/auth/* endpoints
// (register, login, verify, magic, refresh, logout, me, callback, provider) are
// served by dedicated routes — see registerReservedRoutes.
func WithSKAuth(req *RequestContext) (*shttp.Response, error) {
	if req.Host.Config.SKAuth == nil {
		return nil, nil
	}

	injectUserHeaders(req)

	return nil, nil
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

	// An OAuth-issued access token carries an `aud` claim binding it to this
	// environment's protected resource (RFC 8707 / MCP audience binding). Reject
	// one whose audience is not this resource so a token minted for a different
	// resource cannot be replayed here. Session tokens (login/refresh) have no
	// `aud` and skip this check.
	if aud, ok := claims["aud"].(string); ok && aud != "" {
		host := "https://" + req.Host.Name

		if aud != host+req.Host.Config.SKAuth.ResourcePath() && aud != host {
			return
		}
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

	// Only session tokens are refreshable. An OAuth-issued access token carries
	// an `aud` (and often `scope`); refreshing it here would strip that binding
	// and hand back an unrestricted session token, so reject it outright.
	if aud, ok := claims["aud"].(string); ok && aud != "" {
		return &shttp.Response{
			Status: http.StatusUnauthorized,
			Data:   map[string]any{"errors": []string{"token is not refreshable"}},
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
