package hosting

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/html"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type skAuthMiddleware struct {
	req *RequestContext
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

	head := fmt.Sprintf(
		`<script>localStorage.setItem('skauth', JSON.stringify('%s'));window.location.href="%s?verified=true";</script>`,
		data["token"],
		m.req.Host.Config.SKAuth.SuccessURL,
	)

	return m.renderVerifyPage(http.StatusOK, head, ""), nil
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

	// Cross-origin: the POST came from a whitelisted frontend. Stash the token
	// under a one-time code and bounce the user back to that origin's landing
	// page, which injects it into localStorage there (codeLanding). We do NOT
	// touch localStorage here because this host is the wrong origin for the
	// token. SuccessURL is validated at config time to start with '/', so origin
	// (no trailing slash) + successURL joins cleanly.
	if redirect, _ := data["redirect"].(string); redirect != "" {
		landing, err := stashSessionToken(m.req.Context(), redirect, successURL, token)

		if err != nil {
			return m.renderVerifyPage(http.StatusInternalServerError, "", "magic link verification failed"), nil
		}

		head := fmt.Sprintf(`<script>window.location.href=%q;</script>`, landing)

		return m.renderVerifyPage(http.StatusOK, head, ""), nil
	}

	head := fmt.Sprintf(
		`<script>localStorage.setItem('skauth', JSON.stringify(%q));window.location.href=%q;</script>`,
		token,
		successURL+"?verified=true",
	)

	return m.renderVerifyPage(http.StatusOK, head, ""), nil
}

// sessionCode is the payload stored in Redis under the one-time login code. The
// auth host mints the session token and computes the post-login redirect, then
// stashes both so the landing page can inject the token and redirect — without
// the landing host needing its own auth config.
type sessionCode struct {
	Token    string `json:"token"`
	Redirect string `json:"redirect"`
}

// stashSessionToken stores the session token and its post-login redirect under a
// fresh one-time code and returns the landing URL on the destination origin. The
// browser is sent there; codeLanding reads the payload back and injects the token
// into localStorage on that origin. origin is scheme+host (no trailing slash);
// successURL is validated at config time to start with '/'.
func stashSessionToken(ctx context.Context, origin, successURL, token string) (string, error) {
	payload, err := json.Marshal(sessionCode{Token: token, Redirect: origin + successURL})

	if err != nil {
		return "", err
	}

	code := utils.RandomToken(64)

	if err := rediscache.Client().Set(ctx, code, payload, time.Minute*2).Err(); err != nil {
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

	raw, err := rediscache.Client().GetDel(m.req.Context(), code).Result()

	if err != nil || raw == "" {
		return nil
	}

	var sc sessionCode

	if err := json.Unmarshal([]byte(raw), &sc); err != nil {
		return nil
	}

	head := fmt.Sprintf(
		`<script>localStorage.setItem('skauth', JSON.stringify('%s'));window.location.href=%q;</script>`,
		sc.Token,
		sc.Redirect,
	)

	return m.renderVerifyPage(http.StatusOK, head, "")
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

	bearer := user.ParseBearer(req.Header.Get("Authorization"))

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

	bearer := user.ParseBearer(m.req.Header.Get("Authorization"))

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

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   map[string]any{"token": token},
	}, nil
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
