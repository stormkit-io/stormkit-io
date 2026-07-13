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
			return finalizeReserved(req, &shttp.Response{Status: http.StatusNoContent})
		}

		return finalizeReserved(req, fn(&oauthServer{req: req}))
	}
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

// metadataAS serves the RFC 8414 authorization-server metadata document.
func (o *oauthServer) metadataAS() *shttp.Response {
	iss := o.issuer()

	return oauthJSON(http.StatusOK, map[string]any{
		"issuer":                                iss,
		"authorization_endpoint":                iss + "/_stormkit/oauth/authorize",
		"token_endpoint":                        iss + "/_stormkit/oauth/token",
		"registration_endpoint":                 iss + "/_stormkit/oauth/register",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"scopes_supported":                      oauthScopesSupported(),
	})
}

// metadataResource serves the RFC 9728 protected-resource metadata document.
// It points MCP clients at this same host as the authorization server and binds
// the resource identity, which the access-token audience is stamped with.
func (o *oauthServer) metadataResource() *shttp.Response {
	iss := o.issuer()

	return oauthJSON(http.StatusOK, map[string]any{
		"resource":                 iss,
		"authorization_servers":    []string{iss},
		"bearer_methods_supported": []string{"header"},
	})
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
// must sit on an operator-declared AllowedOrigin. Public clients only: we never
// mint a client_secret.
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
// the origin-only redirect check. The AllowedOrigins gate still holds in the
// fallback, so degrading is safe.
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
	}
}

// redirectAllowed reports whether redirectURI is an absolute URL whose origin is
// in the environment's SkAuth AllowedOrigins list. Reusing that list means an
// operator declares the trusted connector origins in one place.
func (o *oauthServer) redirectAllowed(redirectURI string) bool {
	u, err := url.Parse(redirectURI)

	if err != nil || !u.IsAbs() || u.Host == "" {
		return false
	}

	// Host is compared case-insensitively: url.Parse lowercases the scheme but
	// preserves host case, and DNS names are case-insensitive.
	return o.req.Host.Config.SKAuth.IsAllowedOrigin(u.Scheme + "://" + strings.ToLower(u.Host))
}

// authorize serves GET /_stormkit/oauth/authorize — the consent screen. Because
// the SkAuth session lives in localStorage (no server-readable cookie), the page
// is client-assisted: its script reads the token and POSTs back to grant().
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

	return o.consentPage(p, client)
}

// grant serves POST /_stormkit/oauth/authorize. The consent page calls it with
// the user's SkAuth bearer; it validates identity + params and mints a one-time
// authorization code bound to the PKCE challenge.
func (o *oauthServer) grant() *shttp.Response {
	p := o.parseAuthzParams()

	if !o.redirectAllowed(p.redirectURI) {
		return oauthJSON(http.StatusBadRequest, oauthErr("invalid_request", "invalid redirect_uri"))
	}

	if p.responseType != "code" {
		return oauthJSON(http.StatusBadRequest, oauthErr("unsupported_response_type", ""))
	}

	if p.codeChallenge == "" || p.codeChallengeMethod != "S256" {
		return oauthJSON(http.StatusBadRequest, oauthErr("invalid_request", "PKCE S256 required"))
	}

	if client, registered := o.resolveClient(p.clientID); registered && !client.allowsRedirect(p.redirectURI) {
		return oauthJSON(http.StatusBadRequest, oauthErr("invalid_request", "redirect_uri is not registered for this client"))
	}

	claims := user.ParseJWT(&user.ParseJWTArgs{
		Bearer:  user.ParseBearer(o.req.Header.Get("Authorization")),
		Secret:  o.secret(),
		MaxMins: o.req.Host.Config.SKAuth.TTL,
	})

	if claims == nil {
		return oauthJSON(http.StatusUnauthorized, oauthErr("access_denied", "authentication required"))
	}

	uid, _ := claims["uid"].(string)

	if uid == "" {
		return oauthJSON(http.StatusUnauthorized, oauthErr("access_denied", "invalid session"))
	}

	eml, _ := claims["eml"].(string)

	code, err := oauthCodeStore.issue(o.req.Context(), oauthAuthCode{
		UID:           uid,
		EML:           eml,
		ClientID:      p.clientID,
		RedirectURI:   p.redirectURI,
		CodeChallenge: p.codeChallenge,
		Scope:         p.scope,
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
}

// token serves POST /_stormkit/oauth/token — the authorization_code exchange.
// It verifies the PKCE proof and mints an access token in the SkAuth format, so
// the edge validates it like any session token.
func (o *oauthServer) token() *shttp.Response {
	if o.req.PostFormValue("grant_type") != "authorization_code" {
		return oauthJSON(http.StatusBadRequest, oauthErr("unsupported_grant_type", ""))
	}

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

	claims := jwt.MapClaims{"uid": ac.UID, "aud": o.issuer()}

	if ac.EML != "" {
		claims["eml"] = ac.EML
	}

	if ac.Scope != "" {
		claims["scope"] = ac.Scope
	}

	accessToken, err := user.JWT(claims, o.secret())

	if err != nil {
		return oauthJSON(http.StatusInternalServerError, oauthErr("server_error", ""))
	}

	return oauthJSON(http.StatusOK, map[string]any{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   o.tokenTTLMinutes() * 60,
		"scope":        ac.Scope,
	})
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

// consentPage renders the client-assisted consent screen. The script pulls the
// SkAuth token from localStorage and POSTs it back to grant(); when signed out
// it prompts the user to sign in first. html/template escapes the injected
// values in both HTML and JS contexts. A registered client is shown by its
// declared client_name; the requested scopes are listed with human-readable
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
	<div id="skoauth-signedout" class="form-group error" style="display:none">
		You are not signed in. Please sign in to this application in another tab, then reload this page to continue.
	</div>
	<div id="skoauth-actions" class="form-submit" style="display:none">
		<button id="skoauth-allow" class="submit-button">Authorize</button>
		<a id="skoauth-deny" class="secondary" style="margin-left:1rem">Cancel</a>
	</div>
</div>
<script>
(function () {
	var raw = localStorage.getItem('skauth');
	var token = null;
	try { token = raw ? JSON.parse(raw) : null; } catch (e) { token = raw; }

	var actions = document.getElementById('skoauth-actions');
	var signedOut = document.getElementById('skoauth-signedout');

	if (!token) { signedOut.style.display = 'block'; return; }
	actions.style.display = 'block';

	document.getElementById('skoauth-deny').setAttribute('href', "{{ .deny }}");

	document.getElementById('skoauth-allow').addEventListener('click', function () {
		fetch(window.location.pathname + window.location.search, {
			method: 'POST',
			headers: { 'Authorization': 'Bearer ' + token }
		})
			.then(function (r) { return r.json(); })
			.then(function (d) {
				if (d && d.redirect) { window.location.href = d.redirect; }
				else { signedOut.textContent = 'Authorization failed. Please try again.'; signedOut.style.display = 'block'; }
			})
			.catch(function () { signedOut.textContent = 'Authorization failed. Please try again.'; signedOut.style.display = 'block'; });
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
	return &shttp.Response{
		Status: status,
		Headers: shttp.HeadersFromMap(map[string]string{
			"Content-Type":  "application/json",
			"Cache-Control": "no-store",
		}),
		Data: data,
	}
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
