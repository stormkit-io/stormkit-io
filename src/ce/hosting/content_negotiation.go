package hosting

import (
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

			if q, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
				entry.quality = q
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

	if best.specificity < 0 {
		return acceptMatch{specificity: -1}
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
func (p acceptPreference) prefersMarkdown() bool {
	markdown := p.matchFor("text/markdown")

	if markdown.quality <= 0 || markdown.specificity < 0 {
		return false
	}

	html := p.matchFor("text/html")

	if markdown.quality != html.quality {
		return markdown.quality > html.quality
	}

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
