package hosting

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

// markdownCacheTTL bounds how long a converted page is kept.
//
// Correctness does not depend on it: the cache key carries the deployment ID
// and deployments are immutable, so an entry can never describe a page that has
// since changed. It exists only so pages nobody asks for any more stop
// occupying memory.
const markdownCacheTTL = 7 * 24 * time.Hour

// negotiateMarkdown serves the markdown representation of the requested URL, or
// returns nil to let the caller serve the page as it otherwise would.
//
// Two sources, in order. A .md twin the build published is authored content and
// always wins. Conversion of the page's own HTML fills the gap when the build
// published none, and only when the environment asked for it.
func (r *RequestServer) negotiateMarkdown() *shttp.Response {
	if !r.req.Host.Config.Markdown {
		return nil
	}

	// A URL ending in .md names the markdown representation rather than asking
	// for one, so it is answered without consulting Accept and without Vary:
	// it has a single representation, whether the build published it or this
	// request converted it.
	if path.Ext(strings.ToLower(r.req.URL().Path)) == ".md" {
		return r.serveAddressedMarkdown()
	}

	// A deployment that ships a .md twin offers two representations of the same
	// URL, and which one the client gets depends on its Accept header.
	//
	// Vary is set on both representations, not only the markdown one: without
	// it a cache that stored the HTML variant would hand it to the next agent
	// asking for markdown, and the other way round.
	if twin := r.markdownTwin(); twin != "" {
		r.varyAccept = true

		// Negotiation only ever adds a representation. A client that accepts
		// neither type still gets the HTML it would have got before, because
		// refusing a request that used to succeed is a worse answer than
		// serving the page.
		if !r.acceptsMarkdown() {
			return nil
		}

		file := r.req.Host.Config.StaticFiles[twin]

		r.fileMeta = &FileMeta{Name: file.FileName, Headers: file.Headers}
		r.markdown = true

		return r.Static()
	}

	if !r.req.Host.Config.MarkdownConvert {
		return nil
	}

	source := r.fileMeta

	if !isHTMLFile(source) {
		return nil
	}

	// Every HTML page in a converting deployment has a markdown representation,
	// so the URL is negotiable whether or not this request wants one.
	r.varyAccept = true

	if !r.acceptsMarkdown() {
		return nil
	}

	return r.serveConvertedMarkdown(source)
}

// acceptsMarkdown reports whether the client would rather have markdown than
// the page. Every path that offers a markdown representation asks through
// here, so the rule lives in one place.
func (r *RequestServer) acceptsMarkdown() bool {
	return parseAccept(r.req.Header.Get("Accept")).prefersMarkdown()
}

// serveConvertedMarkdown serves the markdown converted from an HTML page, or
// returns nil when the page could not be converted at all — an unreadable
// file, or one over the size caps. The caller then falls back to the HTML for
// a page request, or to the deployment's 404 for a bare .md URL.
func (r *RequestServer) serveConvertedMarkdown(source *FileMeta) *shttp.Response {
	content, ok := r.convertedMarkdown(source)

	if !ok {
		return nil
	}

	r.fileMeta = markdownFileMeta(source, content)
	r.markdown = true

	return r.Static()
}

// serveAddressedMarkdown answers a URL that names the markdown representation
// directly, returning nil to let the caller serve the request as it otherwise
// would.
//
// The file the build published wins, exactly as it does when negotiating. It is
// already resolved by then, so this only has to leave it alone. Converting the
// page it sits beside is the fallback, and only when the environment asked for
// conversion at all.
func (r *RequestServer) serveAddressedMarkdown() *shttp.Response {
	// A .md the build published needs nothing from here: the static handler
	// resolved it like any other file and serves it as itself.
	if r.fileMeta != nil {
		return nil
	}

	if !r.req.Host.Config.MarkdownConvert {
		return nil
	}

	requestPath := strings.ToLower(r.req.URL().Path)
	source := r.staticFile(strings.TrimSuffix(requestPath, ".md"))

	if !isHTMLFile(source) {
		return nil
	}

	return r.serveConvertedMarkdown(source)
}

// convertedMarkdown returns the markdown for an HTML file, converting it on a
// cache miss. ok is false when the page could not be converted — an unreadable
// file, or one over the size caps — and the caller should fall back.
//
// An empty result is a conversion, not a failure: a page with no content in it
// converts to nothing, and that is served as itself.
func (r *RequestServer) convertedMarkdown(source *FileMeta) (content []byte, ok bool) {
	key := r.markdownCacheKey(source)
	ctx := r.req.Context()

	if cached, found := r.cachedMarkdown(key); found {
		return cached, true
	}

	file, err := r.client.GetFile(integrations.GetFileArgs{
		Location:     r.req.Host.Config.StorageLocation,
		DeploymentID: r.req.Host.Config.DeploymentID,
		FileName:     source.Name,
	})

	if err != nil || file == nil {
		return nil, false
	}

	converted, err := htmlToMarkdown(file.Content)

	if err != nil {
		slog.Infof("could not convert %s to markdown: %s", source.Name, err.Error())

		return nil, false
	}

	content = []byte(converted)

	if content == nil {
		content = []byte{}
	}

	if writeErr := r.cache.Set(ctx, key, content, markdownCacheTTL).Err(); writeErr != nil && !errors.Is(writeErr, context.Canceled) {
		slog.Errorf("error caching converted markdown: %s", writeErr.Error())
	}

	return content, true
}

// cachedMarkdown returns the markdown a previous request converted. found is
// false when the cache holds no answer yet.
func (r *RequestServer) cachedMarkdown(key string) (content []byte, found bool) {
	cached, err := r.cache.Get(r.req.Context(), key).Bytes()

	if err != nil {
		if !errors.Is(err, redis.Nil) && !errors.Is(err, context.Canceled) {
			slog.Errorf("error reading converted markdown from cache: %s", err.Error())
		}

		return nil, false
	}

	if cached == nil {
		cached = []byte{}
	}

	return cached, true
}

// markdownCacheKey identifies a converted page. The deployment ID makes it
// immutable, so a new deployment converts afresh rather than inheriting the
// previous one's output.
func (r *RequestServer) markdownCacheKey(source *FileMeta) string {
	return fmt.Sprintf("md:%s:%s", r.req.Host.Config.DeploymentID.String(), source.Name)
}

// markdownFileMeta describes the converted representation of an HTML file.
//
// The ETag is derived from the source page's rather than reused: the two
// representations of a URL must not validate against each other, or a client
// holding one is told its copy of the other is current.
func markdownFileMeta(source *FileMeta, content []byte) *FileMeta {
	original := shttp.HeadersFromMap(source.Headers)
	headers := map[string]string{}

	for _, key := range carriedToMarkdown {
		value := original.Get(key)

		if value == "" {
			continue
		}

		if key == "ETag" {
			value = markdownETag(value)
		}

		headers[key] = value
	}

	return &FileMeta{Name: source.Name, Headers: headers, Body: content}
}

// carriedToMarkdown are the manifest headers that still describe the response
// once its body is markdown rather than the page.
//
// An allowlist and not a list of exclusions: anything describing the bytes —
// Content-Type, Content-Length, and above all Content-Encoding, which a build
// sets when it publishes pre-compressed output — would describe the page and
// not the markdown, and a Content-Encoding that does not match the body is a
// response no client can read.
var carriedToMarkdown = []string{"Cache-Control", "ETag", "Last-Modified"}

// markdownETag turns a page's entity tag into a distinct one for its markdown
// representation, preserving weakness and quoting.
func markdownETag(tag string) string {
	if unquoted, ok := strings.CutSuffix(tag, `"`); ok {
		return unquoted + `-md"`
	}

	if tag == "" {
		return ""
	}

	return tag + "-md"
}

// isHTMLFile reports whether a resolved static file is an HTML page, which is
// the only thing worth converting.
func isHTMLFile(meta *FileMeta) bool {
	if meta == nil {
		return false
	}

	if contentType := shttp.HeadersFromMap(meta.Headers).Get("Content-Type"); contentType != "" {
		return strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "text/html")
	}

	// A manifest entry without a content type is classified by its name, the
	// same way the deployment's own file listing does.
	ext := strings.ToLower(path.Ext(meta.Name))

	return ext == ".html" || ext == ".htm"
}
