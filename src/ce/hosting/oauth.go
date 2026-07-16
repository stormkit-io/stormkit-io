package hosting

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/html"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// oauthCodeStore holds the OAuth authorization codes. The "oauth:code:" prefix
// namespaces them in Redis so they can't collide with the bare one-time login
// codes used by the magic-link landing.
var oauthCodeStore = oneTimeCode{prefix: "oauth:code:", ttl: 5 * time.Minute}

// oauthScope pairs a scope token with the human-readable label shown on the
// consent screen. The catalog is advertised via scopes_supported so clients can
// discover what they may request.
type oauthScope struct {
	Name  string
	Label string
}

// oauthScopeCatalog is the fixed set of scopes this authorization server
// understands. Because the access token only carries identity (uid/eml) and the
// edge injects X-User-Id, scopes are informational — they give the consent
// screen meaningful labels rather than enforcing per-scope access. Unknown
// requested scopes still render, using their raw token as the label.
var oauthScopeCatalog = []oauthScope{
	{Name: "openid", Label: "Verify your identity"},
	{Name: "email", Label: "Access your email address"},
	{Name: "profile", Label: "Access your basic profile information"},
	{Name: "offline_access", Label: "Maintain access when you are offline"},
}

// scopeOfflineAccess is the scope a client requests to receive a refresh token.
// It must be advertised in scopes_supported: Claude only appends it (and thus
// only obtains a refresh token) when the server declares it.
const scopeOfflineAccess = "offline_access"

// scopeRequested reports whether the space-separated scope string contains name.
func scopeRequested(scope, name string) bool {
	for _, f := range strings.Fields(scope) {
		if f == name {
			return true
		}
	}

	return false
}

func oauthScopesSupported() []string {
	names := make([]string, len(oauthScopeCatalog))

	for i, s := range oauthScopeCatalog {
		names[i] = s.Name
	}

	return names
}

// scopeLabels maps a space-separated scope request to the display labels shown
// on consent. A requested scope missing from the catalog falls back to its raw
// token, so the user always sees exactly what was asked for.
func scopeLabels(scope string) []string {
	fields := strings.Fields(scope)
	labels := make([]string, 0, len(fields))

	for _, f := range fields {
		label := f

		for _, s := range oauthScopeCatalog {
			if s.Name == f {
				label = s.Label
				break
			}
		}

		labels = append(labels, label)
	}

	return labels
}

// oauthServer implements the OAuth 2.1 authorization server layered on SkAuth.
// The end user (an app's own SkAuth user) is the resource owner; access tokens
// are signed with the environment's SkAuth secret, so the existing edge
// validation accepts them and injects X-User-Id with no app-side changes.
type oauthServer struct {
	req *RequestContext
}

// serveOAuth wraps an oauthServer handler with the shared gate: when OAuth is
// disabled for the host the path is not reserved, so we fall through to normal
// app serving (this also keeps an app's own /.well-known files reachable).
func serveOAuth(fn func(*oauthServer) *shttp.Response) func(*RequestContext) *shttp.Response {
	return func(req *RequestContext) *shttp.Response {
		if req.Host.Config == nil || !req.Host.Config.SKAuth.OAuthServerEnabled() {
			return HandlerForward(req)
		}

		if req.Method == http.MethodOptions {
			return finalizeReserved(req, withOAuthCORS(&shttp.Response{Status: http.StatusNoContent}))
		}

		return finalizeReserved(req, fn(&oauthServer{req: req}))
	}
}

// oauthCORSHeaders makes the discovery, registration and token endpoints
// reachable from browser-based MCP clients (e.g. MCP Inspector), which probe
// them cross-origin. Those endpoints authenticate via the Authorization header;
// authorize()/grant() instead authenticate via the session cookie, so grant()
// enforces a same-origin Origin check. The wildcard therefore only ever exposes
// the unauthenticated discovery/registration/token responses cross-origin —
// never a cookie-authenticated action.
var oauthCORSHeaders = map[string]string{
	"Access-Control-Allow-Origin":  "*",
	"Access-Control-Allow-Methods": "GET, POST, OPTIONS",
	"Access-Control-Allow-Headers": "Authorization, Content-Type",
}

// withOAuthCORS adds the permissive CORS headers to res, allocating the header
// map if needed. Existing headers are preserved.
func withOAuthCORS(res *shttp.Response) *shttp.Response {
	if res == nil {
		return res
	}

	if res.Headers == nil {
		res.Headers = make(http.Header)
	}

	for k, v := range oauthCORSHeaders {
		res.Headers.Set(k, v)
	}

	return res
}

// OAuth route handlers, wired in registerReservedRoutes and exercised by tests
// through export_test seams.
var (
	handleOAuthMetadataAS       = serveOAuth((*oauthServer).metadataAS)
	handleOAuthMetadataResource = serveOAuth((*oauthServer).metadataResource)
	handleOAuthRegister         = serveOAuth((*oauthServer).register)
	handleOAuthAuthorize        = serveOAuth((*oauthServer).authorize)
	handleOAuthGrant            = serveOAuth((*oauthServer).grant)
	handleOAuthToken            = serveOAuth((*oauthServer).token)
	handleOAuthRevoke           = serveOAuth((*oauthServer).revoke)
)

func (o *oauthServer) secret() string {
	return o.req.Host.Config.SKAuth.Secret
}

// tokenTTLMinutes is the effective access-token lifetime. It mirrors the edge's
// user.ParseJWT default (24h when the env TTL is unset), so the advertised
// expires_in matches how long the token is actually accepted.
func (o *oauthServer) tokenTTLMinutes() int {
	if ttl := o.req.Host.Config.SKAuth.TTL; ttl > 0 {
		return ttl
	}

	return 24 * 60
}

// issuer is the app's own origin; every AS/RS URL hangs off it.
func (o *oauthServer) issuer() string {
	return "https://" + o.req.Host.Name
}

// resourceID is the protected-resource identifier: the issuer plus the
// configured MCP path (e.g. "https://host/mcp"). MCP clients require this to
// match the connector URL the user entered, path included — a bare issuer is
// rejected outright. When no path is configured it degrades to the issuer.
func (o *oauthServer) resourceID() string {
	return o.issuer() + o.req.Host.Config.SKAuth.ResourcePath()
}

// metadataAS serves the RFC 8414 authorization-server metadata document.
func (o *oauthServer) metadataAS() *shttp.Response {
	iss := o.issuer()

	return oauthJSON(http.StatusOK, map[string]any{
		"issuer":                                     iss,
		"authorization_endpoint":                     iss + "/_stormkit/oauth/authorize",
		"token_endpoint":                             iss + "/_stormkit/oauth/token",
		"registration_endpoint":                      iss + "/_stormkit/oauth/register",
		"revocation_endpoint":                        iss + "/_stormkit/oauth/revoke",
		"response_types_supported":                   []string{"code"},
		"grant_types_supported":                      []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":           []string{"S256"},
		"token_endpoint_auth_methods_supported":      []string{"none"},
		"revocation_endpoint_auth_methods_supported": []string{"none"},
		"scopes_supported":                           oauthScopesSupported(),
	})
}

// metadataResource serves the RFC 9728 protected-resource metadata document.
// It points MCP clients at this same host as the authorization server and binds
// the resource identity, which the access-token audience is stamped with.
func (o *oauthServer) metadataResource() *shttp.Response {
	// A per-path probe (/.well-known/oauth-protected-resource/<path>) is only
	// answered when it matches this environment's configured MCP path; any other
	// subpath falls through to normal app serving so the app keeps its own
	// well-known files.
	if sub := o.protectedResourceSubpath(); sub != "" && sub != o.req.Host.Config.SKAuth.ResourcePath() {
		return HandlerForward(o.req)
	}

	return oauthJSON(http.StatusOK, map[string]any{
		"resource":                 o.resourceID(),
		"authorization_servers":    []string{o.issuer()},
		"bearer_methods_supported": []string{"header"},
		"scopes_supported":         oauthScopesSupported(),
	})
}

// protectedResourceSubpath returns the path component the client probed after
// /.well-known/oauth-protected-resource, normalized to a leading slash (e.g.
// "/mcp"), or "" for the bare root document.
func (o *oauthServer) protectedResourceSubpath() string {
	const prefix = "/.well-known/oauth-protected-resource"

	return utils.TrimPath(strings.TrimPrefix(o.req.URL().Path, prefix))
}

// registrationRequest is the subset of RFC 7591 client metadata we honour. MCP
// connectors are public PKCE clients, so we only need their redirect_uris and a
// display name; everything else is validated for compatibility but not stored.
type registrationRequest struct {
	RedirectURIs            []string `json:"redirect_uris"`
	ClientName              string   `json:"client_name"`
	Scope                   string   `json:"scope"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

// Bounds on the unauthenticated registration request. They cap how much an
// anonymous caller can push into Redis under the sliding 30-day TTL: the body
// cap fences off the raw read, and the field caps bound what actually gets
// persisted so a single record can't be inflated with a giant name or thousands
// of redirect_uris.
const (
	maxRegisterBodyBytes = 16 * 1024
	maxRedirectURIs      = 10
	maxClientNameLen     = 256
	maxScopeLen          = 512
)

// register serves POST /_stormkit/oauth/register — Dynamic Client Registration
// (RFC 7591). It is unauthenticated by design (MCP clients self-provision before
// any user is involved), so it is rate-limited per host and every redirect_uri
// must pass redirectAllowed — a curated connector preset or, when opted in, an
// RFC 8252 loopback address. Public clients only: we never mint a client_secret.
func (o *oauthServer) register() *shttp.Response {
	if !oauthClients.registerAllowed(o.req.Context(), o.req.Host.Name, o.req.RemoteIP()) {
		return oauthJSON(http.StatusTooManyRequests, oauthErr("temporarily_unavailable", "registration rate limit exceeded, retry later"))
	}

	if o.req.Request != nil && o.req.Request.Body != nil {
		o.req.Request.Body = http.MaxBytesReader(nil, o.req.Request.Body, maxRegisterBodyBytes)
	}

	var body registrationRequest

	if err := o.req.Post(&body); err != nil {
		return oauthJSON(http.StatusBadRequest, oauthErr("invalid_client_metadata", "request body must be valid JSON within the size limit"))
	}

	if len(body.RedirectURIs) == 0 {
		return oauthJSON(http.StatusBadRequest, oauthErr("invalid_redirect_uri", "at least one redirect_uri is required"))
	}

	if len(body.RedirectURIs) > maxRedirectURIs {
		return oauthJSON(http.StatusBadRequest, oauthErr("invalid_redirect_uri", "too many redirect_uris"))
	}

	if len(body.ClientName) > maxClientNameLen || len(body.Scope) > maxScopeLen {
		return oauthJSON(http.StatusBadRequest, oauthErr("invalid_client_metadata", "client_name or scope is too long"))
	}

	for _, uri := range body.RedirectURIs {
		if !o.redirectAllowed(uri) {
			return oauthJSON(http.StatusBadRequest, oauthErr("invalid_redirect_uri", "every redirect_uri must be on an allowed origin"))
		}
	}

	// We only issue public (PKCE) clients, so a request for a confidential auth
	// method is refused rather than silently downgraded to none.
	if body.TokenEndpointAuthMethod != "" && body.TokenEndpointAuthMethod != "none" {
		return oauthJSON(http.StatusBadRequest, oauthErr("invalid_client_metadata", "only token_endpoint_auth_method=none (public client) is supported"))
	}

	client, err := oauthClients.register(o.req.Context(), oauthClient{
		ClientName:   body.ClientName,
		RedirectURIs: body.RedirectURIs,
		Scope:        body.Scope,
		IssuedAt:     time.Now().Unix(),
	})

	if err != nil {
		return oauthJSON(http.StatusInternalServerError, oauthErr("server_error", ""))
	}

	return oauthJSON(http.StatusCreated, map[string]any{
		"client_id":                  client.ClientID,
		"client_id_issued_at":        client.IssuedAt,
		"redirect_uris":              client.RedirectURIs,
		"client_name":                client.ClientName,
		"scope":                      client.Scope,
		"grant_types":                []string{"authorization_code"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": "none",
	})
}

// resolveClient looks up the Dynamic Client Registration record for clientID.
// It returns (client, true) when a record exists and (zero, false) for an
// unregistered client_id — including on a Redis outage — so callers fall back to
// the origin-only redirect check. The connector-preset (and opt-in loopback) gate
// in redirectAllowed still holds in the fallback, so degrading is safe.
func (o *oauthServer) resolveClient(clientID string) (oauthClient, bool) {
	client, err := oauthClients.get(o.req.Context(), clientID)

	if err != nil {
		return oauthClient{}, false
	}

	return client, true
}

type authzParams struct {
	responseType        string
	clientID            string
	redirectURI         string
	codeChallenge       string
	codeChallengeMethod string
	state               string
	scope               string
	resource            string
}

func (o *oauthServer) parseAuthzParams() authzParams {
	q := o.req.Query()

	return authzParams{
		responseType:        q.Get("response_type"),
		clientID:            q.Get("client_id"),
		redirectURI:         q.Get("redirect_uri"),
		codeChallenge:       q.Get("code_challenge"),
		codeChallengeMethod: q.Get("code_challenge_method"),
		state:               q.Get("state"),
		scope:               q.Get("scope"),
		resource:            q.Get("resource"),
	}
}

// validResource reports whether an RFC 8707 resource indicator names this
// server's protected resource. Only one resource is served per environment, so
// the accepted values are the path-qualified resource id and, tolerantly, the
// bare issuer. An empty resource is treated as "not requested" by callers.
func (o *oauthServer) validResource(resource string) bool {
	return resource == o.resourceID() || resource == o.issuer()
}

// audienceFor resolves the access-token audience (RFC 8707). A client may bind
// the token to a specific resource via the resource indicator; when it does, that
// value is echoed into aud. Otherwise the token is scoped to this environment's
// resource id. The caller has already rejected an invalid requested resource.
func (o *oauthServer) audienceFor(requested, stored string) string {
	return utils.GetString(requested, stored, o.resourceID())
}

// oauthConnectorPresets is the curated allow-list of MCP connector redirect
// origins. These are fixed, public, well-known values published by each vendor,
// and they move over time (ChatGPT migrated its callback path; Claude added
// claude.com alongside claude.ai). Maintaining them here — rather than asking
// every operator to hand-enter them into AllowedOrigins, where they silently rot
// on the next vendor change — means one edit propagates to every environment.
// Redirect matching for unregistered clients is origin-only, so a dynamic
// callback path (e.g. ChatGPT's /connector/oauth/{id}) is covered for free.
//
// Allow-listing an origin grants nothing on its own: every token still requires
// the user to clear the consent screen, and these origins are not
// attacker-controlled.
//
// Last verified: 2026-07-14.
var oauthConnectorPresets = []string{
	"https://claude.ai",
	"https://claude.com",
	"https://chatgpt.com",
}

// loopbackHosts are the RFC 8252 §7.3 loopback IP literals, keyed as
// url.Hostname() reports them (IPv6 brackets stripped). localhost is handled
// separately (see isLoopbackHost) so it matches case-insensitively as a DNS name.
var loopbackHosts = map[string]bool{
	"127.0.0.1": true,
	"::1":       true,
}

// redirectAllowed reports whether redirectURI is an absolute URL permitted to
// receive an authorization code. It trusts the curated connector presets and,
// when the environment opts in, RFC 8252 loopback redirects for native/CLI
// clients. Enabling the OAuth server is itself the opt-in to the presets, so no
// per-origin configuration is required for the supported connectors.
func (o *oauthServer) redirectAllowed(redirectURI string) bool {
	u, err := url.Parse(redirectURI)

	if err != nil || !u.IsAbs() || u.Host == "" {
		return false
	}

	if o.isLoopbackRedirect(u) {
		return true
	}

	// Host is compared case-insensitively: url.Parse lowercases the scheme but
	// preserves host case, and DNS names are case-insensitive.
	origin := u.Scheme + "://" + strings.ToLower(u.Host)

	for _, preset := range oauthConnectorPresets {
		if preset == origin {
			return true
		}
	}

	return false
}

// isLoopbackRedirect reports whether u is a loopback redirect the environment
// has opted in to. Per RFC 8252 §7.3 the port is ignored — the native client
// binds an ephemeral port on the user's machine — so only scheme+host are
// checked here; the caller's exact-match narrowing (for registered clients)
// likewise compares on host, not port.
func (o *oauthServer) isLoopbackRedirect(u *url.URL) bool {
	if !o.req.Host.Config.SKAuth.AllowLoopbackRedirects() {
		return false
	}

	return isLoopbackHost(u.Hostname())
}

// authorize serves GET /_stormkit/oauth/authorize — the consent screen. The
// SkAuth session rides this top-level navigation as a cookie; authorize() reads
// it server-side and renders consent only when a valid session is present,
// delegating to the app's login page otherwise.
func (o *oauthServer) authorize() *shttp.Response {
	p := o.parseAuthzParams()

	// redirect_uri must be validated before we can safely bounce any error to
	// it; an unvalidated redirect is an open-redirect vector.
	if !o.redirectAllowed(p.redirectURI) {
		return o.errorPage("The redirect_uri is not allowed for this application.")
	}

	// A registered client narrows the check to exactly the redirect_uris it
	// registered; an unregistered client_id falls back to the origin-only gate.
	client, registered := o.resolveClient(p.clientID)

	if registered && !client.allowsRedirect(p.redirectURI) {
		return o.errorPage("The redirect_uri is not registered for this client.")
	}

	if p.responseType != "code" {
		return o.redirectError(p, "unsupported_response_type", "only response_type=code is supported")
	}

	if p.codeChallenge == "" || p.codeChallengeMethod != "S256" {
		return o.redirectError(p, "invalid_request", "a PKCE code_challenge using S256 is required")
	}

	// The consent screen must know who is granting access. In cookie mode the
	// session rides this top-level navigation; if it is absent we delegate to
	// the app's own login page, which authenticates the user and bounces back
	// here once the shared session cookie is set. Only then do we render consent.
	if _, _, ok := o.sessionIdentity(); !ok {
		return o.delegateToLogin(p)
	}

	return o.consentPage(p, client)
}

// oauthLoginRetryParam marks a return-from-login navigation. If /authorize is
// reached with it set and there is still no session, the delegation loop is
// broken with an actionable error instead of bouncing to login again.
const oauthLoginRetryParam = "sk_auth_retried"

// sessionIdentity resolves the SkAuth session on the current request (cookie or
// Authorization bearer) into a user id and email. ok is false when there is no
// valid session — the caller then delegates to login.
func (o *oauthServer) sessionIdentity() (uid, eml string, ok bool) {
	claims := user.ParseJWT(&user.ParseJWTArgs{
		Bearer:  sessionBearer(o.req),
		Secret:  o.secret(),
		MaxMins: o.req.Host.Config.SKAuth.TTL,
	})

	if claims == nil {
		return "", "", false
	}

	uid, _ = claims["uid"].(string)

	if uid == "" {
		return "", "", false
	}

	eml, _ = claims["eml"].(string)

	return uid, eml, true
}

// delegateToLogin redirects an unauthenticated /authorize navigation to the
// app's configured login page with a return_to pointing back at this exact
// authorize request (with the retry marker). When no login URL is configured,
// or we have already bounced once, it surfaces an actionable error rather than
// dead-ending or looping.
func (o *oauthServer) delegateToLogin(p authzParams) *shttp.Response {
	loginURL := strings.TrimSpace(o.req.Host.Config.SKAuth.LoginURL)

	if loginURL == "" || o.req.Query().Get(oauthLoginRetryParam) == "1" {
		return o.errorPage("You are not signed in. Sign in to the application and try connecting again.")
	}

	returnTo := appendQuery(o.authorizeURL(), map[string]string{oauthLoginRetryParam: "1"})

	if strings.HasPrefix(loginURL, "/") {
		loginURL = o.issuer() + loginURL
	}

	target := appendQuery(loginURL, map[string]string{"return_to": returnTo})

	return &shttp.Response{
		Status:   http.StatusFound,
		Redirect: &target,
		Headers: shttp.HeadersFromMap(map[string]string{
			"Cache-Control": "no-store",
		}),
	}
}

// authorizeURL reconstructs the absolute URL of the current /authorize request
// so the app can bounce the user back to it after login.
func (o *oauthServer) authorizeURL() string {
	u := *o.req.URL()
	u.Scheme = "https"
	u.Host = o.req.Host.Name

	return u.String()
}

// grant serves POST /_stormkit/oauth/authorize. The consent page calls it with
// the session cookie (credentials:'include'); it validates identity + params and
// mints a one-time authorization code bound to the PKCE challenge.
func (o *oauthServer) grant() *shttp.Response {
	p := o.parseAuthzParams()

	// grant() authenticates from the session cookie, so it must never be drivable
	// cross-site. SameSite=Lax already withholds the cookie on a cross-site POST;
	// this Origin check enforces the same-origin invariant server-side so the
	// guarantee survives any future loosening of the cookie's SameSite policy. The
	// same-origin consent page sends a matching Origin; native/CLI clients use
	// /token, not grant(), and send no Origin here.
	if origin := o.req.Header.Get("Origin"); origin != "" && origin != o.issuer() {
		return oauthJSON(http.StatusForbidden, oauthErr("access_denied", "cross-origin request rejected"))
	}

	if !o.redirectAllowed(p.redirectURI) {
		return oauthJSON(http.StatusBadRequest, oauthErr("invalid_request", "invalid redirect_uri"))
	}

	if p.responseType != "code" {
		return oauthJSON(http.StatusBadRequest, oauthErr("unsupported_response_type", ""))
	}

	if p.codeChallenge == "" || p.codeChallengeMethod != "S256" {
		return oauthJSON(http.StatusBadRequest, oauthErr("invalid_request", "PKCE S256 required"))
	}

	if p.resource != "" && !o.validResource(p.resource) {
		return oauthJSON(http.StatusBadRequest, oauthErr("invalid_target", "the requested resource is not served by this server"))
	}

	if client, registered := o.resolveClient(p.clientID); registered && !client.allowsRedirect(p.redirectURI) {
		return oauthJSON(http.StatusBadRequest, oauthErr("invalid_request", "redirect_uri is not registered for this client"))
	}

	uid, eml, ok := o.sessionIdentity()

	if !ok {
		return oauthJSON(http.StatusUnauthorized, oauthErr("access_denied", "authentication required"))
	}

	code, err := oauthCodeStore.issue(o.req.Context(), oauthAuthCode{
		UID:           uid,
		EML:           eml,
		ClientID:      p.clientID,
		RedirectURI:   p.redirectURI,
		CodeChallenge: p.codeChallenge,
		Scope:         p.scope,
		Resource:      p.resource,
	})

	if err != nil {
		return oauthJSON(http.StatusInternalServerError, oauthErr("server_error", ""))
	}

	return oauthJSON(http.StatusOK, map[string]any{
		"redirect": appendQuery(p.redirectURI, map[string]string{"code": code, "state": p.state}),
	})
}

// oauthAuthCode is the Redis-stashed authorization code payload.
type oauthAuthCode struct {
	UID           string `json:"uid"`
	EML           string `json:"eml"`
	ClientID      string `json:"clientId"`
	RedirectURI   string `json:"redirectUri"`
	CodeChallenge string `json:"codeChallenge"`
	Scope         string `json:"scope"`
	Resource      string `json:"resource,omitempty"`
}

// token serves POST /_stormkit/oauth/token. It dispatches on grant_type: the
// authorization_code exchange (the initial connection) and the refresh_token
// rotation (silent renewal at access-token expiry). Both mint an access token in
// the SkAuth format, so the edge validates it like any session token.
func (o *oauthServer) token() *shttp.Response {
	switch o.req.PostFormValue("grant_type") {
	case "authorization_code":
		return o.tokenAuthorizationCode()
	case "refresh_token":
		return o.tokenRefresh()
	default:
		return oauthJSON(http.StatusBadRequest, oauthErr("unsupported_grant_type", ""))
	}
}

// revoke serves POST /_stormkit/oauth/revoke (RFC 7009). It revokes a refresh
// token by deleting it from the store, killing the connection's ability to renew
// silently. Access tokens are stateless HS256 JWTs with a short TTL, so they are
// not separately revocable and simply expire — this is documented in the AS
// metadata by advertising only revocation of the refresh token. Per RFC 7009 §2.2
// the endpoint returns 200 whether or not the token was valid, so it can't be
// used to probe token validity; token_type_hint is advisory and ignored (we only
// hold refresh tokens).
func (o *oauthServer) revoke() *shttp.Response {
	if token := o.req.PostFormValue("token"); token != "" {
		_ = oauthRefreshTokens.revoke(o.req.Context(), token)
	}

	return withOAuthCORS(&shttp.Response{
		Status: http.StatusOK,
		Headers: shttp.HeadersFromMap(map[string]string{
			"Cache-Control": "no-store",
		}),
	})
}

// tokenAuthorizationCode redeems a PKCE-bound authorization code. When the grant
// carried offline_access it also mints a rotating refresh token so the
// connection survives past the first access-token expiry.
func (o *oauthServer) tokenAuthorizationCode() *shttp.Response {
	code := o.req.PostFormValue("code")
	verifier := o.req.PostFormValue("code_verifier")

	if code == "" || verifier == "" {
		return oauthJSON(http.StatusBadRequest, oauthErr("invalid_request", "code and code_verifier are required"))
	}

	var ac oauthAuthCode

	if err := oauthCodeStore.redeem(o.req.Context(), code, &ac); err != nil {
		if errors.Is(err, errCodeNotFound) {
			return oauthJSON(http.StatusBadRequest, oauthErr("invalid_grant", "authorization code is invalid or expired"))
		}

		return oauthJSON(http.StatusInternalServerError, oauthErr("server_error", ""))
	}

	if o.req.PostFormValue("redirect_uri") != ac.RedirectURI {
		return oauthJSON(http.StatusBadRequest, oauthErr("invalid_grant", "redirect_uri mismatch"))
	}

	// Bind the code to the client_id it was issued for. Without this, origin-level
	// redirect_uri matching alone would let a different client on an allowed
	// origin redeem a code minted for another.
	if o.req.PostFormValue("client_id") != ac.ClientID {
		return oauthJSON(http.StatusBadRequest, oauthErr("invalid_grant", "client_id mismatch"))
	}

	if !verifyPKCE(verifier, ac.CodeChallenge) {
		return oauthJSON(http.StatusBadRequest, oauthErr("invalid_grant", "PKCE verification failed"))
	}

	// The token request may repeat the RFC 8707 resource indicator; if present it
	// must name the same resource the code was issued for.
	requested := o.req.PostFormValue("resource")

	if requested != "" && !o.validResource(requested) {
		return oauthJSON(http.StatusBadRequest, oauthErr("invalid_target", "the requested resource is not served by this server"))
	}

	if requested != "" && ac.Resource != "" && requested != ac.Resource {
		return oauthJSON(http.StatusBadRequest, oauthErr("invalid_grant", "resource does not match the authorization request"))
	}

	aud := o.audienceFor(requested, ac.Resource)

	return o.issueTokens(oauthTokenGrant{
		uid:      ac.UID,
		eml:      ac.EML,
		clientID: ac.ClientID,
		scope:    ac.Scope,
		aud:      aud,
	})
}

// tokenRefresh rotates a refresh token: the presented token is consumed and a
// fresh access/refresh pair is minted from the identity and grant it stood in
// for. An unknown or already-rotated token yields invalid_grant per RFC 6749.
func (o *oauthServer) tokenRefresh() *shttp.Response {
	refresh := o.req.PostFormValue("refresh_token")

	if refresh == "" {
		return oauthJSON(http.StatusBadRequest, oauthErr("invalid_request", "refresh_token is required"))
	}

	// rotate() consumes the presented token atomically (single-use), which is the
	// correct security primitive: a peek-then-delete would open a replay window
	// where two concurrent refreshes both mint. The tradeoff is fail-closed — if
	// re-issue below fails on a transient Redis error the consumed token is gone
	// and the connector re-consents, which is preferable to a replayable token.
	payload, err := oauthRefreshTokens.rotate(o.req.Context(), refresh)

	if err != nil {
		if errors.Is(err, errRefreshNotFound) {
			return oauthJSON(http.StatusBadRequest, oauthErr("invalid_grant", "refresh token is invalid or expired"))
		}

		return oauthJSON(http.StatusInternalServerError, oauthErr("server_error", ""))
	}

	// OAuth 2.1 requires public clients to send client_id; when present it must
	// match the one the refresh token was issued to. We don't hard-require it, so
	// a client that omits it still works — the single-use refresh token is itself
	// the proof of possession.
	if clientID := o.req.PostFormValue("client_id"); clientID != "" && clientID != payload.ClientID {
		return oauthJSON(http.StatusBadRequest, oauthErr("invalid_grant", "client_id mismatch"))
	}

	// A resource indicator on refresh may narrow, but not redirect, the audience.
	if requested := o.req.PostFormValue("resource"); requested != "" {
		if !o.validResource(requested) {
			return oauthJSON(http.StatusBadRequest, oauthErr("invalid_target", "the requested resource is not served by this server"))
		}

		if payload.Audience != "" && requested != payload.Audience {
			return oauthJSON(http.StatusBadRequest, oauthErr("invalid_grant", "resource does not match the original grant"))
		}
	}

	return o.issueTokens(oauthTokenGrant{
		uid:      payload.UID,
		eml:      payload.EML,
		clientID: payload.ClientID,
		scope:    payload.Scope,
		aud:      payload.Audience,
	})
}

// oauthTokenGrant is the identity + grant parameters an access (and optional
// refresh) token is minted from.
type oauthTokenGrant struct {
	uid      string
	eml      string
	clientID string
	scope    string
	aud      string
}

// issueTokens mints the SkAuth-format access token for g and, when the grant
// includes offline_access, a rotating refresh token. The refresh token is always
// re-issued on this path, so a refresh grant rotates it (the presented one was
// already consumed by rotate()).
func (o *oauthServer) issueTokens(g oauthTokenGrant) *shttp.Response {
	claims := jwt.MapClaims{"uid": g.uid, "aud": g.aud}

	if g.eml != "" {
		claims["eml"] = g.eml
	}

	if g.scope != "" {
		claims["scope"] = g.scope
	}

	accessToken, err := user.JWT(claims, o.secret())

	if err != nil {
		return oauthJSON(http.StatusInternalServerError, oauthErr("server_error", ""))
	}

	body := map[string]any{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   o.tokenTTLMinutes() * 60,
		"scope":        g.scope,
	}

	if scopeRequested(g.scope, scopeOfflineAccess) {
		refresh, err := oauthRefreshTokens.issue(o.req.Context(), oauthRefreshPayload{
			UID:      g.uid,
			EML:      g.eml,
			ClientID: g.clientID,
			Scope:    g.scope,
			Audience: g.aud,
		})

		if err != nil {
			return oauthJSON(http.StatusInternalServerError, oauthErr("server_error", ""))
		}

		body["refresh_token"] = refresh
	}

	return oauthJSON(http.StatusOK, body)
}

// verifyPKCE checks the S256 proof: base64url(sha256(verifier)) == challenge.
func verifyPKCE(verifier, challenge string) bool {
	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])

	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}

// redirectError bounces an authorization error back to the (already validated)
// redirect_uri per the OAuth spec, preserving state.
func (o *oauthServer) redirectError(p authzParams, code, desc string) *shttp.Response {
	params := map[string]string{"error": code, "error_description": desc}

	if p.state != "" {
		params["state"] = p.state
	}

	return &shttp.Response{
		Redirect: utils.Ptr(appendQuery(p.redirectURI, params)),
		Status:   http.StatusFound,
	}
}

func (o *oauthServer) errorPage(msg string) *shttp.Response {
	return &shttp.Response{
		Status: http.StatusBadRequest,
		Headers: shttp.HeadersFromMap(map[string]string{
			"Content-Type":            "text/html",
			"Cache-Control":           "no-store",
			"X-Frame-Options":         "DENY",
			"Content-Security-Policy": "frame-ancestors 'none'",
		}),
		Data: html.MustRender(html.RenderArgs{
			PageTitle:   "Stormkit - Authorize",
			PageContent: `<div class="container"><h1>Authorization error</h1><p>{{ .message }}</p></div>`,
			ContentData: map[string]any{"message": msg},
		}),
	}
}

// consentPage renders the consent screen. authorize() has already established a
// session, so the Authorize button simply POSTs back to grant() with the
// session cookie riding the same-origin request (credentials:'include'); grant
// mints the code and returns the redirect target. html/template escapes the
// injected values in both HTML and JS contexts. A registered client is shown by
// its declared client_name; the requested scopes are listed with human-readable
// labels so the user sees exactly what is being granted.
func (o *oauthServer) consentPage(p authzParams, client oauthClient) *shttp.Response {
	name := client.ClientName

	if name == "" {
		name = p.clientID
	}

	if name == "" {
		name = "An application"
	}

	deny := appendQuery(p.redirectURI, map[string]string{"error": "access_denied", "state": p.state})

	content := `
<div class="container">
	<h1>Authorize access</h1>
	<h3><strong>{{ .client }}</strong> is requesting access to your account.</h3>
	{{ if .scopes }}
	<ul class="skoauth-scopes">
		{{ range .scopes }}<li>{{ . }}</li>{{ end }}
	</ul>
	{{ end }}
	<div id="skoauth-error" class="form-group error" style="display:none"></div>
	<div id="skoauth-actions" class="form-submit">
		<button id="skoauth-allow" class="submit-button">Authorize</button>
		<a id="skoauth-deny" class="secondary" style="margin-left:1rem">Cancel</a>
	</div>
</div>
<script>
(function () {
	var err = document.getElementById('skoauth-error');

	function fail() { err.textContent = 'Authorization failed. Please try again.'; err.style.display = 'block'; }

	document.getElementById('skoauth-deny').setAttribute('href', "{{ .deny }}");

	document.getElementById('skoauth-allow').addEventListener('click', function () {
		fetch(window.location.pathname + window.location.search, {
			method: 'POST',
			credentials: 'include'
		})
			.then(function (r) { return r.json(); })
			.then(function (d) {
				if (d && d.redirect) { window.location.href = d.redirect; }
				else { fail(); }
			})
			.catch(fail);
	});
})();
</script>`

	return &shttp.Response{
		Status: http.StatusOK,
		Headers: shttp.HeadersFromMap(map[string]string{
			"Content-Type":            "text/html",
			"Referrer-Policy":         "no-referrer",
			"Cache-Control":           "no-store",
			"X-Frame-Options":         "DENY",
			"Content-Security-Policy": "frame-ancestors 'none'",
		}),
		Data: html.MustRender(html.RenderArgs{
			PageTitle:   "Stormkit - Authorize",
			PageContent: content,
			ContentData: map[string]any{"client": name, "deny": deny, "scopes": scopeLabels(p.scope)},
		}),
	}
}

func oauthJSON(status int, data any) *shttp.Response {
	return withOAuthCORS(&shttp.Response{
		Status: status,
		Headers: shttp.HeadersFromMap(map[string]string{
			"Content-Type":  "application/json",
			"Cache-Control": "no-store",
		}),
		Data: data,
	})
}

func oauthErr(code, desc string) map[string]any {
	out := map[string]any{"error": code}

	if desc != "" {
		out["error_description"] = desc
	}

	return out
}

// appendQuery adds params to a URL, preserving any existing query.
func appendQuery(rawURL string, params map[string]string) string {
	u, err := url.Parse(rawURL)

	if err != nil {
		return rawURL
	}

	q := u.Query()

	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}

	u.RawQuery = q.Encode()

	return u.String()
}
