package hosting

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type HtmlMarkdownSuite struct {
	suite.Suite
}

// convert renders a document body, padding it with enough prose to clear the
// minMarkdownProse floor so a test can assert on the markup it cares about.
func (s *HtmlMarkdownSuite) convert(body string) string {
	const filler = "<p>This paragraph exists so the document carries enough prose to be converted at all.</p>"

	out, err := htmlToMarkdown(htmlToMarkdownParams{
		HTML:     []byte("<html><body>" + body + filler + "</body></html>"),
		BasePath: "/docs/deploying.html",
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

	s.Contains(out, "- [One](/docs/one)")
	s.Contains(out, "- [Two](/docs/two)")
}

func (s *HtmlMarkdownSuite) Test_ResolvesRelativeLinksAndImages() {
	out := s.convert(`
		<a href="../guides/x">Relative</a>
		<a href="https://other.example/abs">Absolute</a>
		<img src="/img/logo.png" alt="Logo">
	`)

	s.Contains(out, "[Relative](/guides/x)")
	s.Contains(out, "[Absolute](https://other.example/abs)")
	s.Contains(out, "![Logo](/img/logo.png)")
}

// The conversion is cached per file, so two request paths that resolve to the
// same file must produce byte-identical output — otherwise whichever converted
// first decides every link for the other.
func (s *HtmlMarkdownSuite) Test_OutputDependsOnFileNotRequest() {
	body := `<html><body><a href="./guide">Guide</a><img src="../img/x.png" alt="X">` +
		`<p>Filler prose so the document clears the minimum prose floor for conversion.</p></body></html>`

	first, err := htmlToMarkdown(htmlToMarkdownParams{HTML: []byte(body), BasePath: "/docs/index.html"})
	s.Require().NoError(err)

	second, err := htmlToMarkdown(htmlToMarkdownParams{HTML: []byte(body), BasePath: "/docs/index.html"})
	s.Require().NoError(err)

	s.Equal(first, second)
	s.Contains(first, "[Guide](/docs/guide)")
	s.Contains(first, "![X](/img/x.png)")
}

// No request header may reach the output: it is cached under a key that carries
// neither the scheme nor the host, so anything request-shaped would be served
// to every later client.
func (s *HtmlMarkdownSuite) Test_NoSchemeOrHostInOutput() {
	out := s.convert(`<a href="/pricing">Pricing</a><img src="logo.png" alt="Logo">`)

	s.NotContains(out, "https://")
	s.NotContains(out, "http://")
	s.Contains(out, "[Pricing](/pricing)")
	s.Contains(out, "![Logo](/docs/logo.png)")
}

// Page content must not be able to break out of the link syntax it is placed
// in: the audience for this output is agents, so an injected second link is a
// prompt-injection primitive rather than a rendering glitch.
func (s *HtmlMarkdownSuite) Test_HrefCannotBreakOutOfLinkSyntax() {
	out := s.convert(`<a href="https://ok.example/a) INJECTED [login](https://evil.example">docs</a>`)

	s.NotContains(out, "INJECTED [login]")
	s.NotContains(out, "](https://evil.example)")
	s.Contains(out, "[docs](https://ok.example/a%29")
}

// Inert markup is never shown to a reader and must not reach the output.
func (s *HtmlMarkdownSuite) Test_DropsInertMarkup() {
	out := s.convert(`<template><p>templated</p></template><noscript>enable js</noscript><p>Shown.</p>`)

	s.Contains(out, "Shown.")
	s.NotContains(out, "templated")
	s.NotContains(out, "enable js")
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

func (s *HtmlMarkdownSuite) Test_InlineCode() {
	out := s.convert("<p>Run <code>npm ci</code> first.</p>")

	s.Contains(out, "`npm ci`")
}

func (s *HtmlMarkdownSuite) Test_Table() {
	out := s.convert(`<table>
		<thead><tr><th>Page</th><th>Size</th></tr></thead>
		<tbody><tr><td>Config</td><td>16 KB</td></tr></tbody>
	</table>`)

	s.Contains(out, "| Page")
	s.Contains(out, "| Size")
	s.Contains(out, "| Config")
	s.Contains(out, "| 16 KB")
}

func (s *HtmlMarkdownSuite) Test_Blockquote() {
	out := s.convert("<blockquote><p>Quoted line.</p></blockquote>")

	s.Contains(out, "> Quoted line.")
}

func (s *HtmlMarkdownSuite) Test_Emphasis() {
	out := s.convert("<p><strong>Bold</strong> and <em>italic</em>.</p>")

	s.Contains(out, "**Bold**")
	s.Contains(out, "*italic*")
}

// A text node following an inline element keeps the space between them, or
// `<b>Config</b> page` renders as one word.
func (s *HtmlMarkdownSuite) Test_KeepsSpaceAfterInlineElements() {
	out := s.convert("<p>Visit <strong>Config</strong> page and run <code>npm ci</code> now.</p>")

	s.Contains(out, "**Config** page")
	s.Contains(out, "`npm ci` now")
}

// Angle brackets are common in prose (breadcrumbs, comparisons) and escaping

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

// A backtick run inside a code block used to widen the fence one character at
// a time, rescanning the whole body each pass. Bounded now, but worth holding.
func (s *HtmlMarkdownSuite) Test_BacktickRunConvertsQuickly() {
	body := "<pre><code>" + strings.Repeat("`", 512<<10) + "</code></pre>"

	start := time.Now()
	_, err := htmlToMarkdown(htmlToMarkdownParams{
		HTML: []byte("<html><body>" + body + "</body></html>"), BasePath: "/x.html",
	})

	s.NoError(err)
	s.Less(time.Since(start), 5*time.Second)
}

// Deeply nested inline elements used to be re-walked once per level.
func (s *HtmlMarkdownSuite) Test_DeeplyNestedInlineConvertsQuickly() {
	body := strings.Repeat("<em>", 400) + strings.Repeat("<br>a", 100_000) + strings.Repeat("</em>", 400)

	start := time.Now()
	_, err := htmlToMarkdown(htmlToMarkdownParams{
		HTML: []byte("<html><body>" + body + "</body></html>"), BasePath: "/x.html",
	})

	s.NoError(err)
	s.Less(time.Since(start), 5*time.Second)
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
