package hosting

import (
	"errors"
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Bounds on what the converter will do for a single request. Conversion runs on
// the edge on a cache miss, so the document, the walk and the output are all
// capped the same way parseAccept caps the Accept header: a hostile or merely
// enormous page must not turn one request into unbounded CPU and allocation.
const (
	maxConvertibleHTML = 4 << 20
	maxMarkdownNodes   = 250_000
	maxMarkdownOutput  = 2 << 20
)

// minMarkdownProse is the number of characters of non-link text a converted
// document must carry to be worth serving.
//
// It rejects the two shapes that convert to something worse than the HTML they
// came from: a client-rendered SPA whose body is an empty root element, and a
// shell that server-renders its navigation and nothing else. Both would
// otherwise hand an agent a document with no content in it.
const minMarkdownProse = 64

// errMarkdownTooLarge is returned when a document exceeds one of the conversion
// bounds. The caller serves the HTML instead.
var errMarkdownTooLarge = errors.New("hosting: document too large to convert")

// skippedElements never contribute text to a markdown representation. Dropping
// them is a category rule rather than a heuristic: none of them is prose, so
// there is no judgement being made about which part of the page is content.
var skippedElements = map[atom.Atom]bool{
	atom.Script:   true,
	atom.Style:    true,
	atom.Noscript: true,
	atom.Template: true,
	atom.Svg:      true,
	atom.Canvas:   true,
	atom.Iframe:   true,
	atom.Object:   true,
	atom.Embed:    true,
	atom.Head:     true,
}

// htmlToMarkdownParams are the arguments of htmlToMarkdown.
type htmlToMarkdownParams struct {
	HTML []byte
	// BaseURL resolves relative hrefs and srcs. Agents routinely pass converted
	// markdown to a later step where the originating URL is gone, so links have
	// to survive on their own.
	BaseURL *url.URL
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

	doc, err := html.Parse(strings.NewReader(string(p.HTML)))

	if err != nil {
		return "", err
	}

	c := &markdownConverter{base: p.BaseURL}

	if err := c.walk(doc); err != nil {
		return "", err
	}

	if c.prose < minMarkdownProse {
		return "", nil
	}

	return strings.TrimSpace(c.out.String()) + "\n", nil
}

// listFrame tracks one level of an open list so nested items indent correctly
// and ordered items keep counting.
type listFrame struct {
	ordered bool
	index   int
}

// markdownConverter walks an HTML tree and emits markdown.
//
// Output is assembled through pending() rather than written directly, so block
// separation is decided once when the next text arrives instead of by every
// element guessing how much whitespace preceded it.
type markdownConverter struct {
	base   *url.URL
	out    strings.Builder
	lists  []listFrame
	quotes int
	// pending is the number of newlines owed before the next write: 1 for a
	// line break, 2 for a block break.
	pending int
	// prose counts characters of text outside anchors, which is what decides
	// whether the document carries content — see minMarkdownProse.
	prose int
	nodes int
	// anchors is the depth of open <a> elements; text inside one is a label
	// rather than prose.
	anchors int
	// pre is true inside <pre>, where whitespace is significant.
	pre bool
}

// lineBreak owes a single newline before the next write.
func (c *markdownConverter) lineBreak() {
	if c.out.Len() > 0 && c.pending < 1 {
		c.pending = 1
	}
}

// blockBreak owes a blank line before the next write.
func (c *markdownConverter) blockBreak() {
	if c.out.Len() > 0 && c.pending < 2 {
		c.pending = 2
	}
}

// prefix is the indentation and quoting that opens a line at the current depth.
func (c *markdownConverter) prefix() string {
	return strings.Repeat("> ", c.quotes) + strings.Repeat("  ", len(c.lists))
}

// write emits text, paying any owed newlines and re-establishing the line
// prefix first.
func (c *markdownConverter) write(s string) {
	if s == "" {
		return
	}

	// The prefix opens a line, and the first line of the document is one too:
	// nothing is owed before it, but a quote or list starting there still needs
	// its marker.
	if c.pending > 0 {
		c.out.WriteString(strings.Repeat("\n", c.pending))
		c.pending = 0
		c.out.WriteString(c.prefix())
	} else if c.out.Len() == 0 {
		c.out.WriteString(c.prefix())
	}

	c.out.WriteString(s)
}

// walk renders a node and its children.
func (c *markdownConverter) walk(n *html.Node) error {
	c.nodes++

	if c.nodes > maxMarkdownNodes || c.out.Len() > maxMarkdownOutput {
		return errMarkdownTooLarge
	}

	switch n.Type {
	case html.TextNode:
		c.text(n.Data)
		return nil
	case html.ElementNode:
		return c.element(n)
	}

	return c.children(n)
}

// children renders every child of a node in order.
func (c *markdownConverter) children(n *html.Node) error {
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if err := c.walk(child); err != nil {
			return err
		}
	}

	return nil
}

// text emits a text node, collapsing whitespace outside <pre> the way a browser
// would and escaping the characters that would otherwise read as markup.
func (c *markdownConverter) text(s string) {
	if c.pre {
		c.write(s)
		return
	}

	collapsed := collapseSpaces(s)

	if collapsed == "" {
		return
	}

	// A leading space separates this text from the inline element before it —
	// `<b>Config</b> page` is two words — so it is kept in the middle of a line
	// and dropped where it would only indent the start of one.
	if strings.HasPrefix(collapsed, " ") && (c.pending > 0 || c.out.Len() == 0) {
		collapsed = collapsed[1:]
	}

	if collapsed == "" {
		return
	}

	escaped := escapeMarkdown(collapsed)

	// A line opening with a block marker would be read as one. Only the first
	// character can do that, and only when this text starts the line.
	if c.pending > 0 || c.out.Len() == 0 {
		if strings.HasPrefix(escaped, ">") || strings.HasPrefix(escaped, "#") {
			escaped = "\\" + escaped
		}
	}

	if c.anchors == 0 {
		c.prose += len(strings.TrimSpace(collapsed))
	}

	c.write(escaped)
}

// element renders a single element.
func (c *markdownConverter) element(n *html.Node) error {
	if skippedElements[n.DataAtom] || isHidden(n) {
		return nil
	}

	switch n.DataAtom {
	case atom.Br:
		c.lineBreak()
		return nil
	case atom.Hr:
		c.blockBreak()
		c.write("---")
		c.blockBreak()
		return nil
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
		return c.heading(n)
	case atom.A:
		return c.anchor(n)
	case atom.Img:
		c.image(n)
		return nil
	case atom.Pre:
		return c.codeBlock(n)
	case atom.Code, atom.Kbd, atom.Samp:
		return c.inlineCode(n)
	case atom.Strong, atom.B:
		return c.wrap(n, "**")
	case atom.Em, atom.I:
		return c.wrap(n, "_")
	case atom.Del, atom.S:
		return c.wrap(n, "~~")
	case atom.Ul, atom.Ol:
		return c.list(n)
	case atom.Li:
		return c.listItem(n)
	case atom.Blockquote:
		return c.blockquote(n)
	case atom.Table:
		return c.table(n)
	}

	if isBlockElement(n.DataAtom) {
		c.blockBreak()

		if err := c.children(n); err != nil {
			return err
		}

		c.blockBreak()

		return nil
	}

	return c.children(n)
}

// heading renders h1-h6 as the matching number of hashes.
func (c *markdownConverter) heading(n *html.Node) error {
	level := int(n.Data[1] - '0')

	c.blockBreak()
	c.write(strings.Repeat("#", level) + " ")

	if err := c.children(n); err != nil {
		return err
	}

	c.blockBreak()

	return nil
}

// anchor renders a link, resolving its target against the base URL. A link with
// no usable target degrades to its own text rather than disappearing.
func (c *markdownConverter) anchor(n *html.Node) error {
	href := c.resolve(attr(n, "href"))

	if href == "" {
		return c.children(n)
	}

	c.write("[")
	c.anchors++

	err := c.children(n)

	c.anchors--

	if err != nil {
		return err
	}

	c.write("](" + href + ")")

	return nil
}

// image renders an image as its alt text and resolved source.
func (c *markdownConverter) image(n *html.Node) {
	src := c.resolve(attr(n, "src"))

	if src == "" {
		return
	}

	alt := attr(n, "alt")

	if alt == "" {
		alt = attr(n, "title")
	}

	c.write("![" + escapeMarkdown(collapseSpaces(alt)) + "](" + src + ")")
}

// codeBlock renders <pre> as a fenced block, taking the language from the
// class of the element or of the <code> inside it.
//
// The body is the element's plain text: syntax-highlighted markup is a tree of
// spans whose only meaningful content is the code itself.
func (c *markdownConverter) codeBlock(n *html.Node) error {
	language := codeLanguage(n)

	if language == "" {
		if inner := firstElement(n, atom.Code); inner != nil {
			language = codeLanguage(inner)
		}
	}

	body := strings.Trim(textContent(n), "\n")

	if strings.TrimSpace(body) == "" {
		return nil
	}

	c.prose += len(strings.TrimSpace(body))

	// A body containing a fence of its own needs a longer one around it.
	fence := "```"

	for strings.Contains(body, fence) {
		fence += "`"
	}

	c.blockBreak()
	c.write(fence + language)

	for _, line := range strings.Split(body, "\n") {
		c.lineBreak()
		c.write(line)
	}

	c.lineBreak()
	c.write(fence)
	c.blockBreak()

	return nil
}

// inlineCode renders <code> outside a <pre> as a span, choosing a backtick run
// long enough to survive the content.
func (c *markdownConverter) inlineCode(n *html.Node) error {
	body := collapseSpaces(textContent(n))

	if body == "" {
		return nil
	}

	if c.anchors == 0 {
		c.prose += len(strings.TrimSpace(body))
	}

	ticks := "`"

	for strings.Contains(body, ticks) {
		ticks += "`"
	}

	pad := ""

	if strings.HasPrefix(body, "`") || strings.HasSuffix(body, "`") {
		pad = " "
	}

	c.write(ticks + pad + body + pad + ticks)

	return nil
}

// wrap renders an element's children between a pair of delimiters, dropping the
// delimiters when the element turns out to be empty.
func (c *markdownConverter) wrap(n *html.Node, delimiter string) error {
	if strings.TrimSpace(textContent(n)) == "" {
		return nil
	}

	c.write(delimiter)

	if err := c.children(n); err != nil {
		return err
	}

	c.write(delimiter)

	return nil
}

// list opens a list level so its items indent and, when ordered, count.
func (c *markdownConverter) list(n *html.Node) error {
	c.blockBreak()
	c.lists = append(c.lists, listFrame{ordered: n.DataAtom == atom.Ol})

	err := c.children(n)

	c.lists = c.lists[:len(c.lists)-1]

	if err != nil {
		return err
	}

	c.blockBreak()

	return nil
}

// listItem renders one item, marked according to the list that encloses it. An
// item outside any list still renders as a bullet rather than losing its text.
func (c *markdownConverter) listItem(n *html.Node) error {
	marker := "- "

	if depth := len(c.lists); depth > 0 {
		frame := &c.lists[depth-1]

		if frame.ordered {
			frame.index++
			marker = itoa(frame.index) + ". "
		}
	}

	c.lineBreak()

	// The marker belongs outside the item's own indentation: prefix() already
	// counts this list level, so the marker is written after the pending
	// newline is paid and before any child block adds to it.
	c.write(marker)

	return c.children(n)
}

// blockquote renders a quoted block, prefixing every line it contains.
func (c *markdownConverter) blockquote(n *html.Node) error {
	c.blockBreak()
	c.quotes++

	err := c.children(n)

	c.quotes--

	if err != nil {
		return err
	}

	c.blockBreak()

	return nil
}

// table renders a GFM table.
//
// The first row becomes the header, because GFM has no way to express a table
// without one and a table whose first row is data still reads correctly.
func (c *markdownConverter) table(n *html.Node) error {
	rows := tableRows(n)

	if len(rows) == 0 {
		return nil
	}

	width := 0

	for _, row := range rows {
		if len(row) > width {
			width = len(row)
		}
	}

	c.blockBreak()

	for index, row := range rows {
		cells := make([]string, width)

		for i := range cells {
			if i < len(row) {
				cells[i] = escapeTableCell(row[i])
				c.prose += len(strings.TrimSpace(row[i]))
			}
		}

		c.lineBreak()
		c.write("| " + strings.Join(cells, " | ") + " |")

		if index == 0 {
			c.lineBreak()
			c.write("|" + strings.Repeat(" --- |", width))
		}
	}

	c.blockBreak()

	return nil
}

// resolve turns a possibly relative reference into an absolute URL.
//
// References that carry their own scheme are left alone, and unparseable ones
// are dropped rather than emitted half-formed.
func (c *markdownConverter) resolve(ref string) string {
	ref = strings.TrimSpace(ref)

	if ref == "" || strings.HasPrefix(ref, "#") {
		return ""
	}

	parsed, err := url.Parse(ref)

	if err != nil {
		return ""
	}

	if c.base == nil || parsed.IsAbs() {
		return ref
	}

	return c.base.ResolveReference(parsed).String()
}

// attr returns an attribute's value, or an empty string when absent.
func attr(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, name) {
			return a.Val
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

// isBlockElement reports whether an element separates the text around it, and
// therefore owes a blank line on both sides.
func isBlockElement(a atom.Atom) bool {
	switch a {
	case atom.P, atom.Div, atom.Section, atom.Article, atom.Main, atom.Nav,
		atom.Header, atom.Footer, atom.Aside, atom.Form, atom.Fieldset,
		atom.Figure, atom.Figcaption, atom.Dl, atom.Dt, atom.Dd, atom.Address,
		atom.Details, atom.Summary, atom.Dialog:
		return true
	}

	return false
}

// collapseSpaces reduces every run of whitespace to a single space, the way a
// browser lays out text outside a <pre>.
func collapseSpaces(s string) string {
	var b strings.Builder

	space := false

	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '\r', '\f', '\v', ' ':
			space = true
		default:
			if space {
				b.WriteByte(' ')
			}

			space = false

			b.WriteRune(r)
		}
	}

	// Leading and trailing runs both survive as a single space: they separate
	// this text node from the inline elements around it, and the caller decides
	// whether a separator that would open a line is worth keeping.
	if space {
		b.WriteByte(' ')
	}

	return b.String()
}

// escapeMarkdown escapes the characters that would otherwise be read as markup
// when the text is parsed back.
//
// Deliberately conservative: over-escaping makes the output unpleasant for the
// humans who will also read it, so only the characters that change meaning
// inline are touched. `>` is not among them — it opens a blockquote only at the
// start of a line, which text() handles where it can see the line.
func escapeMarkdown(s string) string {
	var b strings.Builder

	for _, r := range s {
		switch r {
		case '\\', '`', '*', '_', '[', ']', '<':
			b.WriteByte('\\')
		}

		b.WriteRune(r)
	}

	return b.String()
}

// escapeTableCell renders a cell's text on one line, since a GFM table cell
// cannot contain a line break or a bare pipe.
func escapeTableCell(s string) string {
	return strings.ReplaceAll(escapeMarkdown(collapseSpaces(s)), "|", "\\|")
}

// textContent returns an element's text with markup removed, skipping the
// subtrees that never contribute text.
func textContent(n *html.Node) string {
	var b strings.Builder

	var walk func(*html.Node)

	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
			return
		}

		if node.Type == html.ElementNode {
			if skippedElements[node.DataAtom] || isHidden(node) {
				return
			}

			if node.DataAtom == atom.Br {
				b.WriteByte('\n')
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(n)

	return b.String()
}

// firstElement returns the first descendant with the given tag.
func firstElement(n *html.Node, a atom.Atom) *html.Node {
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.DataAtom == a {
			return child
		}

		if found := firstElement(child, a); found != nil {
			return found
		}
	}

	return nil
}

// codeLanguage extracts the language of a code block from its class list,
// covering the `language-go` and `lang-go` conventions highlighters emit.
func codeLanguage(n *html.Node) string {
	for _, class := range strings.Fields(attr(n, "class")) {
		for _, prefix := range []string{"language-", "lang-"} {
			if name := strings.TrimPrefix(class, prefix); name != class && name != "" {
				return name
			}
		}
	}

	return ""
}

// tableRows collects a table's cells, row by row, in document order. Rows are
// gathered from the whole subtree so thead/tbody/tfoot grouping — which markdown
// has no way to express — does not need handling of its own.
func tableRows(n *html.Node) [][]string {
	rows := [][]string{}

	var walk func(*html.Node)

	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			if skippedElements[node.DataAtom] || isHidden(node) {
				return
			}

			// A nested table is rendered as the text of the cell holding it
			// rather than recursed into: markdown cannot nest tables.
			if node.DataAtom == atom.Tr {
				cells := []string{}

				for cell := node.FirstChild; cell != nil; cell = cell.NextSibling {
					if cell.Type != html.ElementNode {
						continue
					}

					if cell.DataAtom == atom.Td || cell.DataAtom == atom.Th {
						cells = append(cells, textContent(cell))
					}
				}

				if len(cells) > 0 {
					rows = append(rows, cells)
				}

				return
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(n)

	return rows
}

// itoa renders a small positive list index without pulling in strconv for the
// one call site that needs it.
func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}

	return itoa(n/10) + string(rune('0'+n%10))
}
