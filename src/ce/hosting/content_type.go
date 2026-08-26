package hosting

import "strings"

// htmlContentType is the media type every HTML check in hosting agrees on.
const htmlContentType = "text/html"

// isHTMLContentType reports whether a Content-Type value names an HTML
// document. Media types are case-insensitive (RFC 9110 §8.3.1) and header
// values may carry surrounding whitespace, so the value is normalised before
// the prefix test — anything past the type, such as `; charset=utf-8`, is
// irrelevant to the question.
//
// Every HTML decision in hosting goes through here so that snippet injection,
// analytics page views, cache policy and markdown conversion can never disagree
// about the same response.
func isHTMLContentType(contentType string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), htmlContentType)
}
