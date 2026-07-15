package hosting

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// registerReservedRoutes attaches Stormkit's platform auth endpoints (the
// discrete /_stormkit/auth/* handlers) as explicit routes instead of path-prefix
// branches inside WithSKAuth. Each handler is wrapped with WithHost to resolve
// the per-app config and applies finalizeReserved rather than the full
// RequestServer.finalize(): reserved endpoints are platform pages, so customer
// snippets and page-view analytics must not bleed into them.
//
// The cross-cutting parts of the old middleware stay in WithSKAuth: the verified
// bearer -> identity header injection (which runs for every app request) and the
// bare /_stormkit/auth one-time-code landing (which must work even on hosts with
// no auth config of their own).
//
// Ordering matters. gorilla/mux matches in registration order, so these routes
// must be registered before the catch-all in Services(); the dynamic {provider}
// route is registered last so the fixed endpoints above take precedence.
func registerReservedRoutes(s *shttp.Service) {
	sk := s.NewEndpoint("/_stormkit")
	sk.Handler(http.MethodGet, "/analytics.js", WithHost(handleAnalyticsScript))
	// GET is registered too so the handler's own POST-only check returns the 405,
	// matching the pre-route middleware behaviour.
	sk.Handler(http.MethodGet, "/collect", WithHost(handleCollect))
	sk.Handler(http.MethodPost, "/collect", WithHost(handleCollect))

	// OAuth 2.1 authorization server (opt-in per environment). The .well-known
	// discovery documents live off the app root, not under /_stormkit. All fall
	// through to normal app serving when OAuth is disabled for the host.
	wk := s.NewEndpoint("/.well-known")
	wk.Handler(http.MethodGet, "/oauth-authorization-server", WithHost(handleOAuthMetadataAS))
	wk.Handler(http.MethodOptions, "/oauth-authorization-server", WithHost(handleOAuthMetadataAS))
	wk.Handler(http.MethodGet, "/oauth-protected-resource", WithHost(handleOAuthMetadataResource))
	wk.Handler(http.MethodOptions, "/oauth-protected-resource", WithHost(handleOAuthMetadataResource))
	// RFC 9728 path-aware probe: /.well-known/oauth-protected-resource/<path>.
	// The handler answers only when <path> matches the environment's configured
	// MCP path and otherwise falls through to normal app serving.
	wk.Handler(http.MethodGet, "/oauth-protected-resource/{path:.*}", WithHost(handleOAuthMetadataResource))
	wk.Handler(http.MethodOptions, "/oauth-protected-resource/{path:.*}", WithHost(handleOAuthMetadataResource))

	oauth := s.NewEndpoint("/_stormkit/oauth")
	oauth.Handler(http.MethodPost, "/register", WithHost(handleOAuthRegister))
	oauth.Handler(http.MethodOptions, "/register", WithHost(handleOAuthRegister))
	oauth.Handler(http.MethodGet, "/authorize", WithHost(handleOAuthAuthorize))
	oauth.Handler(http.MethodPost, "/authorize", WithHost(handleOAuthGrant))
	oauth.Handler(http.MethodOptions, "/authorize", WithHost(handleOAuthAuthorize))
	oauth.Handler(http.MethodPost, "/token", WithHost(handleOAuthToken))
	oauth.Handler(http.MethodOptions, "/token", WithHost(handleOAuthToken))

	auth := s.NewEndpoint("/_stormkit/auth")

	// Every path is registered for GET, POST and OPTIONS regardless of the
	// method it ultimately accepts: the handlers do their own method validation
	// (returning a JSON 405), and serveReservedAuth answers OPTIONS preflights.
	// This keeps behaviour identical to the old prefix-matching middleware, which
	// never filtered by method at the routing layer.
	add := func(path string, h func(*RequestContext) *shttp.Response) {
		for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodOptions} {
			auth.Handler(method, path, WithHost(h))
		}
	}

	add("/register", handleAuthRegisterLogin)
	add("/login", handleAuthRegisterLogin)
	add("/verify", handleAuthVerify)
	add("/magic", handleAuthMagic)
	add("/refresh", handleAuthRefresh)
	add("/logout", handleAuthLogout)
	add("/me", handleAuthMe)
	add("/callback", handleAuthOAuthCallback)
	add("/{provider}", handleAuthProvider)
}

// serveReservedAuth wraps a skAuthMiddleware handler with the concerns the
// WithSKAuth chain used to apply around every /_stormkit/auth/* branch: fall
// through to normal app serving when the host has no auth configured, answer
// CORS preflights, inject the verified-bearer identity headers, and finalize
// with platform headers only.
func serveReservedAuth(fn func(*skAuthMiddleware) (*shttp.Response, error)) func(*RequestContext) *shttp.Response {
	return func(req *RequestContext) *shttp.Response {
		// No auth configured -> this path is not reserved for the host; serve the
		// app exactly as the old middleware's nil return did.
		if req.Host.Config.SKAuth == nil {
			return HandlerForward(req)
		}

		if req.Method == http.MethodOptions {
			return finalizeReserved(req, &shttp.Response{Status: http.StatusNoContent})
		}

		injectUserHeaders(req)

		resp, err := fn(&skAuthMiddleware{req: req})

		if err != nil {
			return NewRequestServer(req).Error(err)
		}

		return finalizeReserved(req, resp)
	}
}

var (
	handleAuthVerify        = serveReservedAuth((*skAuthMiddleware).handleVerify)
	handleAuthRefresh       = serveReservedAuth((*skAuthMiddleware).handleRefresh)
	handleAuthLogout        = serveReservedAuth((*skAuthMiddleware).handleLogout)
	handleAuthMe            = serveReservedAuth((*skAuthMiddleware).handleMe)
	handleAuthMagic         = serveReservedAuth((*skAuthMiddleware).handleMagic)
	handleAuthOAuthCallback = serveReservedAuth((*skAuthMiddleware).handleOAuthCallback)

	handleAuthRegisterLogin = serveReservedAuth(func(m *skAuthMiddleware) (*shttp.Response, error) {
		return m.handleRegisterLogin(m.req.URL().Path)
	})

	// handleAuthProvider serves /_stormkit/auth/{provider}. A known provider
	// starts the OAuth flow; anything else falls to the one-time-code landing's
	// terminal state, matching the old middleware fall-through.
	handleAuthProvider = serveReservedAuth(func(m *skAuthMiddleware) (*shttp.Response, error) {
		if provider, ok := isOAuthProviderPath(m.req.URL().Path); ok {
			return m.handleOAuthInitiate(provider)
		}

		return m.handleCallback()
	})
)

// finalizeReserved is the reserved-endpoint counterpart to
// RequestServer.finalize. It applies the platform response headers
// (x-sk-version, Server, and the user-configured CustomHeaders that carry CORS)
// but deliberately skips snippet injection and page-view analytics, which
// belong to customer traffic only.
func finalizeReserved(req *RequestContext, res *shttp.Response) *shttp.Response {
	if res == nil || req.Host == nil || req.Host.Config == nil {
		return res
	}

	return injectHeaders(req, res)
}
