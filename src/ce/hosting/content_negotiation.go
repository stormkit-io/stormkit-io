package hosting

import (
	"math"
	"path"
	"strconv"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
)

// MarkdownContentType is what a markdown representation is served as. RFC 7763
// registers text/markdown; the charset removes any doubt about the encoding.
const MarkdownContentType = "text/markdown; charset=utf-8"

// acceptedType is one entry of a parsed Accept header.
type acceptedType struct {
	mime    string
	quality float64
}

// acceptPreference is a parsed Accept header, answering the only three
// questions the static handler asks of it.
type acceptPreference struct {
	types []acceptedType
	// empty is true when the request carried no usable Accept header at all, in
	// which case RFC 9110 says everything is acceptable.
	empty bool
}

// parseAccept parses an Accept header into its media types and q-values.
// Malformed entries are skipped rather than failing the request: a browser with
// a broken header must still get its page.
func parseAccept(header string) acceptPreference {
	pref := acceptPreference{empty: true}

	for _, part := range strings.Split(header, ",") {
		fields := strings.Split(strings.TrimSpace(part), ";")
		mime := strings.ToLower(strings.TrimSpace(fields[0]))

		if mime == "" {
			continue
		}

		entry := acceptedType{mime: mime, quality: 1}

		for _, param := range fields[1:] {
			key, value, found := strings.Cut(strings.TrimSpace(param), "=")

			if !found || strings.ToLower(strings.TrimSpace(key)) != "q" {
				continue
			}

			// Clamped, and NaN dropped, because the q decides which file is
			// served and the header is whatever the client sent. An out-of-range
			// q used to be harmless — the old comparison took the highest q
			// across every matching range, so an illegal one lifted both types
			// equally — but a q now applies to one type and not the other, and
			// `text/html, */*;q=5` would hand markdown to a client that named
			// HTML at full weight.
			//
			// A q that does not parse at all keeps the default of 1 rather than
			// dropping to 0: a browser sending a truncated `text/html;q=` must
			// still get its page.
			if q, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && !math.IsNaN(q) {
				entry.quality = math.Min(math.Max(q, 0), 1)
			}
		}

		pref.empty = false
		pref.types = append(pref.types, entry)
	}

	return pref
}

// acceptMatch is how a media type matched an Accept header: the q-value that
// applies to it, and how specifically the client named it.
type acceptMatch struct {
	quality float64
	// specificity is 2 for an exact `type/subtype`, 1 for `type/*`, 0 for
	// `*/*`, and -1 when nothing matched.
	specificity int
}

// specificityOf reports how precisely a media range names a media type, or -1
// when the range does not cover it at all.
func specificityOf(entry, mime, family string) int {
	switch entry {
	case mime:
		return 2
	case family + "/*":
		return 1
	case "*/*":
		return 0
	}

	return -1
}

// matchFor returns the q-value a client assigned to a media type, taken from
// the most specific range that covers it.
//
// Most specific and not the highest q across every matching range, per RFC 9110
// §12.5.1: in `text/html;q=0.1, */*` the client downweighted HTML and the
// wildcard does not undo that.
func (p acceptPreference) matchFor(mime string) acceptMatch {
	if p.empty {
		return acceptMatch{quality: 1, specificity: 0}
	}

	family, _, _ := strings.Cut(mime, "/")
	best := acceptMatch{specificity: -1}

	for _, entry := range p.types {
		specificity := specificityOf(entry.mime, mime, family)

		if specificity < 0 {
			continue
		}

		// A more specific range replaces a less specific one outright; among
		// equally specific ranges the client's highest q wins.
		if specificity > best.specificity ||
			(specificity == best.specificity && entry.quality > best.quality) {
			best = acceptMatch{quality: entry.quality, specificity: specificity}
		}
	}

	return best
}

// prefersMarkdown reports whether the client would rather have markdown than
// HTML.
//
// A tie on q-value goes to whichever type the client named more specifically,
// so `text/markdown, */*` — an agent asking for markdown but taking anything —
// gets markdown, while a bare `*/*` from curl still gets the page. A tie at
// equal specificity goes to HTML.
//
// Listing order is not a signal, per RFC 9110 §12.5.1, so `text/markdown,
// text/html` is a tie and serves HTML. A client that wants markdown while
// naming HTML too has to say so with a q-value.
func (p acceptPreference) prefersMarkdown() bool {
	markdown := p.matchFor("text/markdown")

	// Covers "not matched" as well: an unmatched type carries a zero quality.
	if markdown.quality <= 0 {
		return false
	}

	html := p.matchFor("text/html")

	if markdown.quality != html.quality {
		return markdown.quality > html.quality
	}

	// Reachable only with both qualities equal and markdown's above zero, so
	// HTML matched too and its specificity is a real rank rather than the
	// no-match sentinel.
	return markdown.specificity > html.specificity
}

// markdownTwinParams are the arguments of markdownTwin.
type markdownTwinParams struct {
	RequestPath string
	Files       appconf.StaticFileConfig
}

// markdownTwin returns the deployment file holding the markdown representation
// of a request path, or an empty string when the deployment ships none.
//
// Only files the build actually published are considered, so content
// negotiation can never invent a URL: /docs/foo is negotiable exactly when the
// deployment contains /docs/foo.md (or /docs/foo/index.md).
func markdownTwin(p markdownTwinParams) string {
	clean := strings.ToLower(p.RequestPath)
	ext := path.Ext(clean)

	// Assets carry an extension of their own and can never have a twin. They are
	// the bulk of the requests a deployment serves, so they leave before this
	// function allocates anything.
	if ext != "" && ext != ".md" && ext != ".html" {
		return ""
	}

	trimmed := strings.TrimSuffix(clean, "/")
	candidates := []string{}

	switch ext {
	case ".md":
		candidates = append(candidates, clean)
	case ".html":
		candidates = append(candidates, strings.TrimSuffix(clean, ".html")+".md")
	default:
		if trimmed != "" {
			candidates = append(candidates, trimmed+".md")
		}

		candidates = append(candidates, path.Join(clean, "index.md"))
	}

	for _, candidate := range candidates {
		if p.Files[candidate] != nil {
			return candidate
		}
	}

	return ""
}
