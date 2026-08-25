package hosting

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type HtmlMarkdownSuite struct {
	suite.Suite
}

// convert renders a document body, padding it with enough prose to clear the
// minMarkdownProse floor so a test can assert on the markup it cares about.
func (s *HtmlMarkdownSuite) convert(body string) string {
	const filler = "<p>This paragraph exists so the document carries enough prose to be converted at all.</p>"

	base, err := url.Parse("https://example.com/docs/deploying")
	s.Require().NoError(err)

	out, err := htmlToMarkdown(htmlToMarkdownParams{
		HTML:    []byte("<html><body>" + body + filler + "</body></html>"),
		BaseURL: base,
	})

	s.Require().NoError(err)

	return out
}

func (s *HtmlMarkdownSuite) Test_Headings() {
	out := s.convert("<h1>Deploying</h1><h3>Prerequisites</h3>")

	s.Contains(out, "# Deploying")
	s.Contains(out, "### Prerequisites")
}

func (s *HtmlMarkdownSuite) Test_DropsNonContentElements() {
	out := s.convert(`
		<script>var secret = 1;</script>
		<style>.a { color: red }</style>
		<noscript>Enable JavaScript</noscript>
		<template><p>unused</p></template>
		<p>Visible.</p>
	`)

	s.Contains(out, "Visible.")
	s.NotContains(out, "secret")
	s.NotContains(out, "color: red")
	s.NotContains(out, "Enable JavaScript")
	s.NotContains(out, "unused")
}

func (s *HtmlMarkdownSuite) Test_DropsHiddenElements() {
	out := s.convert(`
		<div hidden><p>hidden attribute</p></div>
		<div aria-hidden="true"><p>aria hidden</p></div>
		<div style="display: none"><p>styled away</p></div>
		<p>Shown.</p>
	`)

	s.Contains(out, "Shown.")
	s.NotContains(out, "hidden attribute")
	s.NotContains(out, "aria hidden")
	s.NotContains(out, "styled away")
}

func (s *HtmlMarkdownSuite) Test_NavBecomesLinkList() {
	out := s.convert(`<nav><ul>
		<li><a href="/docs/one">One</a></li>
		<li><a href="/docs/two">Two</a></li>
	</ul></nav>`)

	s.Contains(out, "- [One](https://example.com/docs/one)")
	s.Contains(out, "- [Two](https://example.com/docs/two)")
}

func (s *HtmlMarkdownSuite) Test_ResolvesRelativeLinksAndImages() {
	out := s.convert(`
		<a href="../guides/x">Relative</a>
		<a href="https://other.example/abs">Absolute</a>
		<img src="/img/logo.png" alt="Logo">
	`)

	s.Contains(out, "[Relative](https://example.com/guides/x)")
	s.Contains(out, "[Absolute](https://other.example/abs)")
	s.Contains(out, "![Logo](https://example.com/img/logo.png)")
}

func (s *HtmlMarkdownSuite) Test_OrderedListCounts() {
	out := s.convert("<ol><li>First</li><li>Second</li><li>Third</li></ol>")

	s.Contains(out, "1. First")
	s.Contains(out, "2. Second")
	s.Contains(out, "3. Third")
}

func (s *HtmlMarkdownSuite) Test_FencedCodeKeepsLanguageAndBody() {
	out := s.convert(`<pre><code class="language-go">func main() {
	println("hi")
}</code></pre>`)

	s.Contains(out, "```go")
	s.Contains(out, `func main() {`)
	s.Contains(out, `println("hi")`)
}

// Highlighted code is a tree of spans whose only meaningful content is the code
// itself, and docs are exactly where this matters most.
func (s *HtmlMarkdownSuite) Test_FencedCodeFlattensHighlightSpans() {
	out := s.convert(`<pre><code class="language-bash">` +
		`<span class="tok-k">npm</span> <span class="tok-s">install</span>` +
		`</code></pre>`)

	s.Contains(out, "```bash")
	s.Contains(out, "npm install")
	s.NotContains(out, "tok-k")
	s.NotContains(out, "<span")
}

func (s *HtmlMarkdownSuite) Test_CodeBodyIsNotEscaped() {
	out := s.convert("<pre><code>a * b _ c [d]</code></pre>")

	s.Contains(out, "a * b _ c [d]")
	s.NotContains(out, `\*`)
}

func (s *HtmlMarkdownSuite) Test_InlineCode() {
	out := s.convert("<p>Run <code>npm ci</code> first.</p>")

	s.Contains(out, "`npm ci`")
}

func (s *HtmlMarkdownSuite) Test_Table() {
	out := s.convert(`<table>
		<thead><tr><th>Page</th><th>Size</th></tr></thead>
		<tbody><tr><td>Config</td><td>16 KB</td></tr></tbody>
	</table>`)

	s.Contains(out, "| Page | Size |")
	s.Contains(out, "| --- | --- |")
	s.Contains(out, "| Config | 16 KB |")
}

func (s *HtmlMarkdownSuite) Test_Blockquote() {
	out := s.convert("<blockquote><p>Quoted line.</p></blockquote>")

	s.Contains(out, "> Quoted line.")
}

func (s *HtmlMarkdownSuite) Test_Emphasis() {
	out := s.convert("<p><strong>Bold</strong> and <em>italic</em>.</p>")

	s.Contains(out, "**Bold**")
	s.Contains(out, "_italic_")
}

// A text node following an inline element keeps the space between them, or
// `<b>Config</b> page` renders as one word.
func (s *HtmlMarkdownSuite) Test_KeepsSpaceAfterInlineElements() {
	out := s.convert("<p>Visit <strong>Config</strong> page and run <code>npm ci</code> now.</p>")

	s.Contains(out, "**Config** page")
	s.Contains(out, "`npm ci` now")
}

// Angle brackets are common in prose (breadcrumbs, comparisons) and escaping
// every one of them makes the output unreadable.
func (s *HtmlMarkdownSuite) Test_DoesNotEscapeAngleBracketsInline() {
	out := s.convert("<p>Go to <strong>App</strong> > <strong>Config</strong>.</p>")

	s.Contains(out, "**App** > **Config**")
	s.NotContains(out, "\\>")
}

// A line that opens with one, though, would be read as a quote.
func (s *HtmlMarkdownSuite) Test_EscapesBlockMarkerOpeningALine() {
	out := s.convert("<p>&gt; not a quote</p>")

	s.Contains(out, "\\> not a quote")
}

func (s *HtmlMarkdownSuite) Test_EscapesMarkdownCharactersInProse() {
	out := s.convert("<p>Use a * for wildcards and _ for spaces.</p>")

	s.Contains(out, `\*`)
	s.Contains(out, `\_`)
}

// A client-rendered SPA shell converts to nothing. Serving that to an agent is
// worse than serving the HTML, so the converter must refuse it.
func (s *HtmlMarkdownSuite) Test_EmptyShellIsRefused() {
	out, err := htmlToMarkdown(htmlToMarkdownParams{
		HTML: []byte(`<html><body><div id="root"></div><script src="/app.js"></script></body></html>`),
	})

	s.NoError(err)
	s.Equal("", out)
}

// A shell that server-renders only its navigation has plenty of text and no
// content. Link labels do not count as prose, so it is refused too.
func (s *HtmlMarkdownSuite) Test_NavOnlyShellIsRefused() {
	out, err := htmlToMarkdown(htmlToMarkdownParams{
		HTML: []byte(`<html><body><nav>
			<a href="/one">Getting started</a>
			<a href="/two">Deploying your first app</a>
			<a href="/three">Environments and configuration</a>
		</nav><div id="root"></div></body></html>`),
	})

	s.NoError(err)
	s.Equal("", out)
}

func (s *HtmlMarkdownSuite) Test_OversizedDocumentIsRefused() {
	_, err := htmlToMarkdown(htmlToMarkdownParams{
		HTML: []byte("<html><body>" + strings.Repeat("a", maxConvertibleHTML) + "</body></html>"),
	})

	s.ErrorIs(err, errMarkdownTooLarge)
}

// The node budget has to stop a deeply nested document before the walk does,
// the same way parseAccept bounds a hostile Accept header.
func (s *HtmlMarkdownSuite) Test_NodeBudgetIsEnforced() {
	body := strings.Repeat("<p>x</p>", maxMarkdownNodes)

	_, err := htmlToMarkdown(htmlToMarkdownParams{
		HTML: []byte("<html><body>" + body + "</body></html>"),
	})

	s.ErrorIs(err, errMarkdownTooLarge)
}

func (s *HtmlMarkdownSuite) Test_RealPageShapeIsSmallerThanItsHTML() {
	page := `<html><head><title>Deploying</title><style>body{margin:0}</style></head>
	<body>
		<nav><a href="/docs">Docs</a></nav>
		<main>
			<h1>Deploying</h1>
			<p>Stormkit builds your application and serves the output from the edge.</p>
			<pre><code class="language-bash">stormkit deploy</code></pre>
		</main>
		<script>console.log("analytics")</script>
	</body></html>`

	out := s.convert(page)

	s.Contains(out, "# Deploying")
	s.Contains(out, "```bash")
	s.NotContains(out, "analytics")
	s.NotContains(out, "margin:0")
	s.Less(len(out), len(page))
}

func TestHtmlMarkdownSuite(t *testing.T) {
	suite.Run(t, new(HtmlMarkdownSuite))
}
