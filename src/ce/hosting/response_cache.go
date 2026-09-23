package hosting

import (
	"bytes"
	"compress/gzip"
	"container/list"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"weak"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const (
	defaultResponseCacheBudget  = 64 << 20 // compressed bytes held
	defaultResponseCacheMaxBody = 8 << 20  // body size before compression

	// minCompressibleBody is the smallest body worth compressing. Below it the
	// gzip header and trailer cost more than the compression saves.
	minCompressibleBody = 1024
)

// compressibleTypes are the content types where gzip earns its keep. Images,
// video and modern font formats are already compressed, so a second pass burns
// processor time and occasionally makes them larger.
var compressibleTypes = []string{
	"text/",
	"application/javascript",
	"application/x-javascript",
	"application/json",
	"application/ld+json",
	"application/xml",
	"application/rss+xml",
	"application/atom+xml",
	"application/wasm",
	"image/svg+xml",
}

// responses holds finished static responses for this hosting process.
var responses = responseCacheFromEnv()

// responseCache keeps finished static responses, compressed, so a repeat
// request for the same URL skips reading the file, injecting snippets and
// compressing the body.
//
// Entries are tied to the host configuration they were built from. The config
// is replaced whenever it is reloaded, which happens on every change that can
// alter a response (snippets, headers, a new deployment), so a stale entry
// misses without anyone having to invalidate it. Explicit invalidation only
// frees the memory sooner.
//
// The config is referenced weakly: an entry nobody asks for again must not keep
// a replaced config, with its whole file manifest, alive until it is evicted.
type responseCache struct {
	mu      sync.Mutex
	budget  int64
	maxBody int64
	used    int64
	entries map[string]*list.Element
	order   *list.List // front is most recently used
	writers sync.Pool
}

type responseCacheParams struct {
	Budget  int64
	MaxBody int64
}

type responseCacheEntry struct {
	key      string
	hostName string
	config   weak.Pointer[appconf.Config]
	headers  http.Header
	body     []byte // gzip
	size     int64  // body length before compression
	cost     int64  // memory counted against the budget
}

type responseLookup struct {
	Key    string
	Config *appconf.Config
}

func newResponseCache(p responseCacheParams) *responseCache {
	return &responseCache{
		budget:  p.Budget,
		maxBody: p.MaxBody,
		entries: map[string]*list.Element{},
		order:   list.New(),
		writers: sync.Pool{
			New: func() any {
				writer, _ := gzip.NewWriterLevel(nil, gzip.DefaultCompression)
				return writer
			},
		},
	}
}

// responseCacheFromEnv sizes the cache from STORMKIT_RESPONSE_CACHE_BYTES. Zero
// disables it, which is the way to turn it off without a release.
func responseCacheFromEnv() *responseCache {
	budget := int64(defaultResponseCacheBudget)

	if value := os.Getenv("STORMKIT_RESPONSE_CACHE_BYTES"); value != "" {
		budget = int64(utils.StringToInt(value))
	}

	return newResponseCache(responseCacheParams{Budget: budget, MaxBody: defaultResponseCacheMaxBody})
}

func (rc *responseCache) enabled() bool {
	return rc.budget > 0
}

// get returns the entry for a key, provided it was built from the given config.
// An entry from an older config is dropped on the way.
func (rc *responseCache) get(p responseLookup) (*responseCacheEntry, bool) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	element, ok := rc.entries[p.Key]

	if !ok {
		return nil, false
	}

	entry := element.Value.(*responseCacheEntry)

	if entry.config != weak.Make(p.Config) {
		rc.remove(element)
		return nil, false
	}

	rc.order.MoveToFront(element)

	return entry, true
}

// put stores an entry, evicting least recently used ones to stay inside the
// budget.
func (rc *responseCache) put(entry *responseCacheEntry) {
	size := entry.cost

	if size > rc.budget {
		return
	}

	rc.mu.Lock()
	defer rc.mu.Unlock()

	if element, ok := rc.entries[entry.key]; ok {
		rc.remove(element)
	}

	for rc.used+size > rc.budget {
		oldest := rc.order.Back()

		if oldest == nil {
			return
		}

		rc.remove(oldest)
	}

	rc.entries[entry.key] = rc.order.PushFront(entry)
	rc.used += size
}

// dropHosts removes every entry for a host name the pattern matches. It takes
// the same pattern the host config cache is invalidated with.
func (rc *responseCache) dropHosts(pattern *regexp.Regexp) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	for _, element := range rc.entries {
		if pattern.MatchString(element.Value.(*responseCacheEntry).hostName) {
			rc.remove(element)
		}
	}
}

func (rc *responseCache) clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.entries = map[string]*list.Element{}
	rc.order.Init()
	rc.used = 0
}

// remove must be called with the lock held.
func (rc *responseCache) remove(element *list.Element) {
	entry := element.Value.(*responseCacheEntry)

	rc.used -= entry.cost
	rc.order.Remove(element)
	delete(rc.entries, entry.key)
}

func (rc *responseCache) compressible(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))

	for _, candidate := range compressibleTypes {
		if strings.HasPrefix(ct, candidate) {
			return true
		}
	}

	return false
}

// compress gzips a body. It reports false when the body does not come out
// smaller, since it is then better served as it is.
func (rc *responseCache) compress(body []byte) ([]byte, bool) {
	var buf bytes.Buffer

	writer := rc.writers.Get().(*gzip.Writer)
	defer rc.writers.Put(writer)

	writer.Reset(&buf)

	if _, err := writer.Write(body); err != nil {
		return nil, false
	}

	if err := writer.Close(); err != nil {
		return nil, false
	}

	if buf.Len() >= len(body) {
		return nil, false
	}

	return buf.Bytes(), true
}

// response builds a response from the entry. Headers are copied because the
// response writer adds to them.
func (e *responseCacheEntry) response() *shttp.Response {
	return &shttp.Response{
		Status:  http.StatusOK,
		Headers: e.headers.Clone(),
		Data:    e.body,
	}
}

// cacheKey returns the key a static response to this request is cached under,
// or an empty string when the response must neither be served from the cache
// nor stored in it.
//
// It is computed after the auth wall and redirect rules have run, so nothing is
// ever served from the cache to a request those would have stopped, and the
// rewritten path is known.
func (r *RequestServer) cacheKey() string {
	cnf := r.req.Host.Config

	if !responses.enabled() {
		return ""
	}

	if method := r.req.Method; method != "" && method != http.MethodGet {
		return ""
	}

	// A conditional request is answered with a 304 and no body, which is cheap
	// already and depends on the validator the client holds.
	if r.req.Header.Get("If-None-Match") != "" || r.req.Header.Get("If-Modified-Since") != "" {
		return ""
	}

	// Image resizing picks its output from the query, which the key leaves out.
	if r.req.Query().Has("size") {
		return ""
	}

	if r.hasPerRequestSnippet() {
		return ""
	}

	// A page filled by a loader is as current as its document. The loader keeps
	// its own TTL cache; this one holds finished bodies with no expiry of their
	// own, so it would outlive the data it rendered.
	if r.req.Loader != nil {
		return ""
	}

	// The same URL serves the page or its markdown depending on Accept.
	variant := "page"

	if cnf.Markdown && r.acceptsMarkdown() {
		variant = "markdown"
	}

	// Both paths: snippets match on the original one and files resolve from the
	// rewritten one. The length prefix keeps two different pairs from joining
	// into the same string.
	return fmt.Sprintf(
		"%s|%s|%s|%d:%s|%s",
		r.req.Host.Name,
		cnf.DeploymentID.String(),
		variant,
		len(r.req.OriginalPath),
		r.req.OriginalPath,
		r.req.URL().Path,
	)
}

// hasPerRequestSnippet reports whether a snippet interpolates the request id,
// which makes every response to this host different.
func (r *RequestServer) hasPerRequestSnippet() bool {
	for _, snippet := range r.req.Host.Config.Snippets {
		if snippet.Interpolate && strings.Contains(snippet.Content, appconf.SnippetVarRequestID) {
			return true
		}
	}

	return false
}

// cachedResponse returns the cached response for this request, or nil when it
// has to be built.
func (r *RequestServer) cachedResponse() *shttp.Response {
	if !acceptsGzip(r.req.Header.Get("Accept-Encoding")) {
		return nil
	}

	key := r.cacheKey()

	if key == "" {
		return nil
	}

	entry, ok := responses.get(responseLookup{Key: key, Config: r.req.Host.Config})

	if !ok {
		return nil
	}

	r.servedFromCache = true
	r.bodySize = entry.size
	r.res = entry.response()

	return r.res
}

// cacheResponse stores a finished static response and returns the compressed
// copy to send, so the body is not compressed a second time on the way out.
//
// Only a client that accepts gzip stores one. Any other client could never be
// served the copy, so compressing for it would be paid on every request.
func (r *RequestServer) cacheResponse(res *shttp.Response) *shttp.Response {
	if r.servedFromCache || !acceptsGzip(r.req.Header.Get("Accept-Encoding")) || !r.cacheable(res) {
		return res
	}

	key := r.cacheKey()

	if key == "" {
		return res
	}

	body, _ := responseBytes(res)
	packed, ok := responses.compress(body)

	if !ok {
		return res
	}

	headers := res.Headers.Clone()
	headers.Del("Content-Length")
	headers.Set("Content-Encoding", "gzip")

	// The compression middleware normally stamps this. It is bypassed for an
	// already-encoded response, so a shared cache would otherwise be free to
	// hand these bytes to a client that cannot read them.
	appendVary(headers, "Accept-Encoding")

	entry := &responseCacheEntry{
		key:      key,
		hostName: r.req.Host.Name,
		config:   weak.Make(r.req.Host.Config),
		headers:  headers,
		body:     packed,
		size:     int64(len(body)),
		cost:     int64(len(packed)+len(key)+len(r.req.Host.Name)) + headersSize(headers),
	}

	responses.put(entry)

	r.bodySize = entry.size

	return entry.response()
}

// cacheable reports whether a finished response is a static file worth
// keeping compressed.
func (r *RequestServer) cacheable(res *shttp.Response) bool {
	if res == nil || res.Status != http.StatusOK || r.fileMeta == nil || r.fnInvoked {
		return false
	}

	if len(res.Cookies) > 0 || res.ServeContent != nil || res.Redirect != nil || res.Headers == nil {
		return false
	}

	// A build can publish files that are compressed already; those are served
	// as they are.
	if res.Headers.Get("Content-Encoding") != "" || !responses.compressible(res.Headers.Get("Content-Type")) {
		return false
	}

	body, ok := responseBytes(res)

	return ok && len(body) >= minCompressibleBody && int64(len(body)) <= responses.maxBody
}
