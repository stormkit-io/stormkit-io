package hosting

import (
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
		if !parseAccept(r.req.Header.Get("Accept")).prefersMarkdown() {
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

	source, addressed := r.convertibleSource()

	if source == nil {
		return nil
	}

	// A URL that ends in .md names the markdown representation outright, so it
	// is not negotiable and carries no Vary. Every other path is the page, and
	// only an Accept header moves it off HTML.
	if !addressed && !parseAccept(r.req.Header.Get("Accept")).prefersMarkdown() {
		// The page still needs Vary if a markdown representation of it exists,
		// but only if one does: a deployment whose pages all decline — every
		// client-rendered app — would otherwise advertise a second
		// representation it never has, and Static() drops If-Modified-Since
		// handling on any URL carrying Vary.
		//
		// Converting here to find out would spend a parse on every browser
		// request, so this asks only what the cache already knows. The first
		// markdown request records the answer, and pages carry Vary from then
		// on.
		r.varyAccept = r.hasConvertedMarkdown(source)

		return nil
	}

	content := r.convertedMarkdown(source)

	// Conversion declined — an empty shell, an oversized document, unreadable
	// storage. Returning nil serves the HTML for a page request, and falls
	// through to the deployment's 404 for a bare .md URL the build never
	// shipped.
	if content == nil {
		return nil
	}

	if !addressed {
		r.varyAccept = true
	}

	r.fileMeta = markdownFileMeta(source)
	r.markdown = true
	r.markdownBody = content

	return r.Static()
}

// convertibleSource returns the HTML file whose markdown representation the
// request is asking for, and whether the URL addressed that representation
// directly rather than negotiating for it.
//
// Only static HTML converts. A page rendered by a server function is produced
// per request and never reaches the manifest, so there is nothing here to
// convert — those deployments keep serving HTML until they ship a twin.
func (r *RequestServer) convertibleSource() (*FileMeta, bool) {
	requestPath := strings.ToLower(r.req.URL().Path)

	// The .md form of a page is fetchable in its own right once conversion is
	// enabled: enabling it is a request for that representation to exist, and
	// an agent handed a link to it must be able to follow it.
	if path.Ext(requestPath) == ".md" {
		source := r.staticFile(strings.TrimSuffix(requestPath, ".md"))

		if isHTMLFile(source) {
			return source, true
		}

		return nil, false
	}

	if isHTMLFile(r.fileMeta) {
		return r.fileMeta, false
	}

	return nil, false
}

// convertedMarkdown returns the markdown for an HTML file, converting it on a
// cache miss. A nil return means the page has no markdown representation worth
// serving and the caller should fall back.
func (r *RequestServer) convertedMarkdown(source *FileMeta) []byte {
	key := r.markdownCacheKey(source)
	ctx := r.req.Context()

	if cached, known := r.cachedMarkdown(key); known {
		return cached
	}

	file, err := r.client.GetFile(integrations.GetFileArgs{
		Location:     r.req.Host.Config.StorageLocation,
		DeploymentID: r.req.Host.Config.DeploymentID,
		FileName:     source.Name,
	})

	if err != nil || file == nil {
		return nil
	}

	converted, err := htmlToMarkdown(htmlToMarkdownParams{
		HTML:     file.Content,
		BasePath: source.Name,
	})

	if err != nil {
		slog.Infof("could not convert %s to markdown: %s", source.Name, err.Error())
	}

	if writeErr := r.cache.Set(ctx, key, converted, markdownCacheTTL).Err(); writeErr != nil {
		slog.Errorf("error caching converted markdown: %s", writeErr.Error())
	}

	if converted == "" {
		return nil
	}

	return []byte(converted)
}

// cachedMarkdown reports what the cache knows about a page: its markdown, or
// nil for a page recorded as not worth converting. known is false when the
// cache holds no answer yet.
//
// An empty cached value is that recorded refusal, kept so a shell that cannot
// convert is not re-parsed on every request.
func (r *RequestServer) cachedMarkdown(key string) (content []byte, known bool) {
	cached, err := r.cache.Get(r.req.Context(), key).Result()

	if err != nil {
		if !errors.Is(err, redis.Nil) {
			slog.Errorf("error reading converted markdown from cache: %s", err.Error())
		}

		return nil, false
	}

	if cached == "" {
		return nil, true
	}

	return []byte(cached), true
}

// hasConvertedMarkdown reports whether the cache already holds a markdown
// representation of a page, without producing one.
func (r *RequestServer) hasConvertedMarkdown(source *FileMeta) bool {
	content, known := r.cachedMarkdown(r.markdownCacheKey(source))

	return known && content != nil
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
func markdownFileMeta(source *FileMeta) *FileMeta {
	headers := map[string]string{}

	for key, value := range source.Headers {
		if !carriedToMarkdown[strings.ToLower(strings.TrimSpace(key))] {
			continue
		}

		if strings.EqualFold(key, "ETag") {
			value = markdownETag(value)
		}

		headers[key] = value
	}

	return &FileMeta{Name: source.Name, Headers: headers}
}

// carriedToMarkdown are the manifest headers that still describe the response
// once its body is markdown rather than the page.
//
// An allowlist and not a list of exclusions: anything describing the bytes —
// Content-Type, Content-Length, and above all Content-Encoding, which a build
// sets when it publishes pre-compressed output — would describe the page and
// not the markdown, and a Content-Encoding that does not match the body is a
// response no client can read.
var carriedToMarkdown = map[string]bool{
	"cache-control": true,
	"etag":          true,
	"last-modified": true,
}

// markdownETag turns a page's entity tag into a distinct one for its markdown
// representation, preserving weakness and quoting.
func markdownETag(tag string) string {
	if tag == "" {
		return ""
	}

	if strings.HasSuffix(tag, `"`) {
		return strings.TrimSuffix(tag, `"`) + `-md"`
	}

	return tag + "-md"
}

// isHTMLFile reports whether a resolved static file is an HTML page, which is
// the only thing worth converting.
func isHTMLFile(meta *FileMeta) bool {
	if meta == nil {
		return false
	}

	for key, value := range meta.Headers {
		if strings.EqualFold(key, "Content-Type") {
			return strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), "text/html")
		}
	}

	// A manifest entry without a content type is classified by its name, the
	// same way the deployment's own file listing does.
	ext := strings.ToLower(path.Ext(meta.Name))

	return ext == ".html" || ext == ".htm"
}
