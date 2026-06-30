package hosting

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/lib/pool"
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
