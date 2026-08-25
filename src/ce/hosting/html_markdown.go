package hosting

import (
	"bytes"
	"errors"
	"path"
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// maxConvertibleHTML bounds the document a single request will convert.
//
// Conversion runs on the edge on a cache miss, so the input is capped the same
// way parseAccept caps the Accept header. The largest page this platform's own
// documentation publishes is a little over 200KB, so a megabyte leaves room for
// a genuinely large page while keeping a hostile one from setting the cost.
const maxConvertibleHTML = 1 << 20

// maxMarkdownOutput bounds what a conversion may produce. Markup that is cheap
// to convert can still expand — a page of nothing but backticks converts to
// several times its own size — and the result is cached, so the output is
// capped as well as the input.
const maxMarkdownOutput = 2 << 20

// minMarkdownProse is the number of characters of non-link text a converted
// document must carry to be worth serving.
//
// It rejects the two shapes that convert to something worse than the HTML they
// came from: a client-rendered SPA whose body is an empty root element, and a
// shell that server-renders its navigation and nothing else. Both would
// otherwise hand an agent a document with no content in it.
const minMarkdownProse = 64

// errMarkdownTooLarge is returned when a document exceeds maxConvertibleHTML.
// The caller serves the HTML instead.
var errMarkdownTooLarge = errors.New("hosting: document too large to convert")

// markdownConverter renders HTML as CommonMark. Constructing it walks the
// plugin registry, so one instance is shared across requests — it holds no
// per-conversion state.
var markdownConverter = converter.NewConverter(
	converter.WithPlugins(
		base.NewBasePlugin(),
		commonmark.NewCommonmarkPlugin(),
		table.NewTablePlugin(),
	),
)

// inertElements hold markup that is never rendered to a reader, which the
// converter emits as text because it only skips script and style.
var inertElements = map[atom.Atom]bool{
	atom.Template: true,
	atom.Noscript: true,
}

// htmlToMarkdownParams are the arguments of htmlToMarkdown.
type htmlToMarkdownParams struct {
	HTML []byte
	// BasePath is the path of the file being converted, used to resolve its
	// relative hrefs and srcs into root-relative ones.
	//
	// The file's own path and not the request's: the conversion is cached per
	// file, so anything the output depends on has to be a property of the file.
	// Resolving against the request would let /docs and /docs/ produce different
	// links for the same cache entry, and would put the request's host and
	// scheme — neither of which the cache key carries — into every link.
	BasePath string
}

// htmlToMarkdown converts an HTML document into a markdown representation of
// the same page.
//
// The whole document is converted rather than an article extracted from it.
// Picking out "the content" means guessing which subtree matters, and a wrong
// guess silently drops the page; converting everything is deterministic, and
// the navigation it keeps is useful to an agent working its way through a site.
//
// An empty string is returned when the result carries too little prose to be
// worth serving — see minMarkdownProse.
func htmlToMarkdown(p htmlToMarkdownParams) (string, error) {
	if len(p.HTML) > maxConvertibleHTML {
		return "", errMarkdownTooLarge
	}

	doc, err := html.Parse(bytes.NewReader(p.HTML))

	if err != nil {
		return "", err
	}

	prepared := &documentPreparer{base: path.Dir("/" + strings.TrimPrefix(p.BasePath, "/"))}

	prepared.prepare(doc)

	if prepared.prose < minMarkdownProse {
		return "", nil
	}

	converted, err := markdownConverter.ConvertNode(doc)

	if err != nil {
		return "", err
	}

	if len(converted) > maxMarkdownOutput {
		return "", errMarkdownTooLarge
	}

	trimmed := strings.TrimSpace(string(converted))

	if trimmed == "" {
		return "", nil
	}

	return trimmed + "\n", nil
}

// documentPreparer rewrites a parsed document in place before it is converted,
// doing the two things the converter does not: dropping content the page hides,
// and making relative references resolve away from the page they came from.
//
// It also measures the prose it walks past, so the decision not to convert a
// contentless shell costs nothing beyond the pass that was happening anyway.
type documentPreparer struct {
	base string
	// prose counts characters of text outside anchors. Link labels are excluded
	// so a shell that server-renders only its navigation reads as contentless.
	prose int
}

// prepare walks a node and its children, removing hidden elements, resolving
// references, and accumulating prose.
func (d *documentPreparer) prepare(n *html.Node) {
	child := n.FirstChild

	for child != nil {
		// Held before the node can be detached, or the walk loses its place.
		next := child.NextSibling

		switch {
		case child.Type == html.TextNode:
			if n.DataAtom != atom.A && n.DataAtom != atom.Script && n.DataAtom != atom.Style {
				d.prose += len(strings.TrimSpace(child.Data))
			}
		case child.Type == html.ElementNode && (inertElements[child.DataAtom] || isHidden(child)):
			// Neither is part of the page. The converter drops script, style
			// and comments on its own; this covers the inert markup it keeps
			// and whatever an attribute or inline style hides from a reader.
			n.RemoveChild(child)
		default:
			if child.Type == html.ElementNode {
				d.resolveReferences(child)
			}

			d.prepare(child)
		}

		child = next
	}
}

// resolveReferences rewrites an element's href and src so the markdown still
// points at the right place once it is read somewhere other than this page.
func (d *documentPreparer) resolveReferences(n *html.Node) {
	for i, a := range n.Attr {
		if !strings.EqualFold(a.Key, "href") && !strings.EqualFold(a.Key, "src") {
			continue
		}

		if resolved := d.resolve(a.Val); resolved != "" {
			n.Attr[i].Val = resolved
		}
	}
}

// resolve turns a relative reference into a root-relative one. References that
// already resolve on their own are returned unchanged, as an empty string.
func (d *documentPreparer) resolve(ref string) string {
	trimmed := strings.TrimSpace(ref)

	if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "/") {
		return ""
	}

	// A scheme or a protocol-relative host means the reference names its own
	// origin and must not be rewritten onto ours.
	if strings.HasPrefix(trimmed, "//") || schemeOf(trimmed) != "" {
		return ""
	}

	suffix := ""

	if cut := strings.IndexAny(trimmed, "?#"); cut != -1 {
		suffix = trimmed[cut:]
		trimmed = trimmed[:cut]
	}

	if trimmed == "" {
		return ""
	}

	return path.Join(d.base, trimmed) + suffix
}

// schemeOf returns the URL scheme a reference declares, or an empty string when
// it declares none. A colon appearing after a slash is part of a path rather
// than a scheme separator.
func schemeOf(ref string) string {
	for i := 0; i < len(ref); i++ {
		switch c := ref[i]; {
		case c == ':':
			return ref[:i]
		case c == '/' || c == '?' || c == '#':
			return ""
		}
	}

	return ""
}

// isHidden reports whether an element is hidden from readers. Hidden content is
// not part of the page, so it is dropped for the same reason a <script> is.
func isHidden(n *html.Node) bool {
	for _, a := range n.Attr {
		switch strings.ToLower(a.Key) {
		case "hidden":
			return true
		case "aria-hidden":
			if strings.EqualFold(strings.TrimSpace(a.Val), "true") {
				return true
			}
		case "style":
			style := strings.ToLower(strings.ReplaceAll(a.Val, " ", ""))

			if strings.Contains(style, "display:none") || strings.Contains(style, "visibility:hidden") {
				return true
			}
		}
	}

	return false
}
