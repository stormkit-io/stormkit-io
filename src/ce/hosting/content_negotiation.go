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

// qualityFor returns the q-value the client assigned to a media type, honouring
// the `type/*` and `*/*` wildcards. It returns 0 when the type is not accepted.
func (p acceptPreference) qualityFor(mime string) float64 {
	if p.empty {
		return 1
	}

	family, _, _ := strings.Cut(mime, "/")
	best := 0.0

	for _, entry := range p.types {
		switch entry.mime {
		case mime, family + "/*", "*/*":
			if entry.quality > best {
				best = entry.quality
			}
		}
	}

	return best
}

// prefersMarkdown reports whether the client would rather have markdown than
// HTML. A tie goes to HTML, so a browser sending `*/*` keeps its page.
func (p acceptPreference) prefersMarkdown() bool {
	markdown := p.qualityFor("text/markdown")

	return markdown > 0 && markdown > p.qualityFor("text/html")
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
