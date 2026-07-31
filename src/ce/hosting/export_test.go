package hosting

import (
	"context"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/lib/pool"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// AnalyticsRecord exposes analyticsRecord so tests can assert which requests are
// (and are not) recorded as page views.
func AnalyticsRecord(req *RequestContext, res *shttp.Response) *analytics.Record {
	return analyticsRecord(req, res)
}

// ResolveAnalyticsScript exposes resolveAnalyticsScript so tests can assert the
// override-else-embedded selection without a database.
func ResolveAnalyticsScript(cfg admin.InstanceConfig) ([]byte, string) {
	return resolveAnalyticsScript(cfg)
}

// InjectSnippets exposes injectSnippets for tests.
func InjectSnippets(req *RequestContext, res *shttp.Response) *shttp.Response {
	return injectSnippets(req, res)
}

// InjectOAuthChallenge exposes injectOAuthChallenge for tests.
func InjectOAuthChallenge(req *RequestContext, res *shttp.Response) *shttp.Response {
	return injectOAuthChallenge(req, res)
}

// HandleAuthVerify exposes the reserved /_stormkit/auth/verify route handler.
// Reserved endpoints are unexported handlers now that they are routes rather
// than middleware branches; this seam lets the existing tests drive the handler
// directly without standing up the mux router.
func HandleAuthVerify(req *RequestContext) *shttp.Response {
	return handleAuthVerify(req)
}

// HandleAnalyticsScript exposes the reserved /_stormkit/analytics.js route
// handler to the external hosting_test package.
func HandleAnalyticsScript(req *RequestContext) *shttp.Response {
	return handleAnalyticsScript(req)
}

// OAuth route handler seams for the external hosting_test package.
func HandleOAuthMetadataAS(req *RequestContext) *shttp.Response { return handleOAuthMetadataAS(req) }
func HandleOAuthMetadataResource(req *RequestContext) *shttp.Response {
	return handleOAuthMetadataResource(req)
}
func HandleOAuthRegister(req *RequestContext) *shttp.Response  { return handleOAuthRegister(req) }
func HandleOAuthAuthorize(req *RequestContext) *shttp.Response { return handleOAuthAuthorize(req) }
func HandleOAuthGrant(req *RequestContext) *shttp.Response     { return handleOAuthGrant(req) }
func HandleOAuthToken(req *RequestContext) *shttp.Response     { return handleOAuthToken(req) }
func HandleOAuthRevoke(req *RequestContext) *shttp.Response    { return handleOAuthRevoke(req) }

// OAuthRegisterRateMax exposes the per-host registration limit so the rate-limit
// test can drive it up to the threshold without hard-coding the constant.
func OAuthRegisterRateMax() int { return registerRateMax }

// ResetOAuthRateLimit clears the registration counters for a host (across every
// caller-IP window) so tests start from a clean slate regardless of prior runs
// against the shared Redis.
func ResetOAuthRateLimit(host string) {
	c := rediscache.Client()

	keys, _ := c.Keys(context.Background(), "oauth:reg:rate:"+host+":*")

	if len(keys) > 0 {
		c.Del(context.Background(), keys...)
	}
}

// ServeAuth dispatches a /_stormkit/auth/* request to its route handler, the
// same way registerReservedRoutes wires them into mux. It mirrors the WithSKAuth
// (*shttp.Response, error) signature — always a nil error — so the terminal
// endpoint tests migrate from WithSKAuth with a plain rename.
func ServeAuth(req *RequestContext) (*shttp.Response, error) {
	switch req.URL().Path {
	case "/_stormkit/auth/register", "/_stormkit/auth/login":
		return handleAuthRegisterLogin(req), nil
	case "/_stormkit/auth/verify":
		return handleAuthVerify(req), nil
	case "/_stormkit/auth/magic":
		return handleAuthMagic(req), nil
	case "/_stormkit/auth/refresh":
		return handleAuthRefresh(req), nil
	case "/_stormkit/auth/logout":
		return handleAuthLogout(req), nil
	case "/_stormkit/auth/me":
		return handleAuthMe(req), nil
	case "/_stormkit/auth/token":
		return handleAuthNativeToken(req), nil
	case skauth.CallbackPath:
		return handleAuthOAuthCallback(req), nil
	default:
		return handleAuthProvider(req), nil
	}
}

// EmbeddedScriptETag exposes the embedded default's ETag for tests.
func EmbeddedScriptETag() string {
	return embeddedScriptETag
}

// WaitArtifacts blocks until all in-flight artifacts goroutines have finished
// pushing into the Batcher. Tests use it to drain prior requests before swapping
// the global Batcher, so a leaked push can't land in the next test's buffer.
func WaitArtifacts() {
	artifactsWG.Wait()
}

// ResetBatcher swaps the global Batcher under the same lock Queue uses, avoiding
// a data race between a test's reset and a concurrent push.
func ResetBatcher(b *pool.Buffer) {
	mu.Lock()
	defer mu.Unlock()
	Batcher = b
}

// SetFetchConfigFn replaces the function used to fetch app config from the
// database. It is intended for use in tests only.
func SetFetchConfigFn(fn func(string) ([]*appconf.Config, error)) {
	fetchConfigFn = fn
}

// ResetFetchConfigFn restores the default database-backed fetch function.
func ResetFetchConfigFn() {
	fetchConfigFn = appconf.FetchConfig
}

// InvalidateAppCache removes a hostname entry from the in-process cache,
// allowing tests to force a cold-cache scenario without restarting the process.
func InvalidateAppCache(hostName string) {
	appCacheMu.Lock()
	delete(appCache, hostName)
	appCacheMu.Unlock()
}
