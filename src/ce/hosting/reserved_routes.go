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

	// Each endpoint is registered for the methods it actually accepts; the router
	// returns 405 for anything else, so the handlers no longer self-check. The XHR
	// endpoints (register, login, refresh, me, and magic's POST) also register
	// OPTIONS so a cross-origin (decoupled frontend) credentialed preflight is
	// answered — serveReservedAuth returns 204 for it.
	reg := func(path string, h func(*RequestContext) *shttp.Response, methods ...string) {
		for _, method := range methods {
			auth.Handler(method, path, WithHost(h))
		}
	}

	reg("/register", handleAuthRegisterLogin, http.MethodPost, http.MethodOptions)
	reg("/login", handleAuthRegisterLogin, http.MethodPost, http.MethodOptions)
	reg("/refresh", handleAuthRefresh, http.MethodPost, http.MethodOptions)
	reg("/me", handleAuthMe, http.MethodGet, http.MethodOptions)
	// Magic link: GET follows the emailed verification link, POST requests one.
	reg("/magic", handleAuthMagic, http.MethodGet, http.MethodPost, http.MethodOptions)
	// Navigation endpoints reached by a top-level GET redirect.
	reg("/verify", handleAuthVerify, http.MethodGet)
	reg("/callback", handleAuthOAuthCallback, http.MethodGet)
	reg("/{provider}", handleAuthProvider, http.MethodGet)
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
	handleAuthMe            = serveReservedAuth((*skAuthMiddleware).handleMe)
	handleAuthMagic         = serveReservedAuth((*skAuthMiddleware).handleMagic)
	handleAuthOAuthCallback = serveReservedAuth((*skAuthMiddleware).handleOAuthCallback)

	handleAuthRegisterLogin = serveReservedAuth(func(m *skAuthMiddleware) (*shttp.Response, error) {
		return m.handleRegisterLogin(m.req.URL().Path)
	})

	// handleAuthProvider serves /_stormkit/auth/{provider}. A known provider
	// starts the OAuth flow; anything else is not a reserved endpoint, so it
	// falls through to normal app serving.
	handleAuthProvider = serveReservedAuth(func(m *skAuthMiddleware) (*shttp.Response, error) {
		if provider, ok := isOAuthProviderPath(m.req.URL().Path); ok {
			return m.handleOAuthInitiate(provider)
		}

		return HandlerForward(m.req), nil
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
