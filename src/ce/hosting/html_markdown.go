package hosting

import (
	"bytes"
	"fmt"

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

// htmlToMarkdown converts an HTML document into a markdown representation of
// the same page.
//
// The whole document is converted rather than an article extracted from it.
// Picking out "the content" means guessing which subtree matters, and a wrong
// guess silently drops the page; converting everything is deterministic, and
// the navigation it keeps is useful to an agent working its way through a site.
//
// A page with nothing in it converts to nothing, and that is the answer served:
// conversion is opt-in, so a deployment that turns it on and gets an empty
// document has learned something true about its pages.
func htmlToMarkdown(document []byte) (string, error) {
	if len(document) > maxConvertibleHTML {
		return "", fmt.Errorf("hosting: document is %d bytes, over the %d byte conversion limit", len(document), maxConvertibleHTML)
	}

	doc, err := html.Parse(bytes.NewReader(document))

	if err != nil {
		return "", err
	}

	dropInertElements(doc)

	converted, err := markdownConverter.ConvertNode(doc)

	if err != nil {
		return "", err
	}

	if len(converted) > maxMarkdownOutput {
		return "", fmt.Errorf("hosting: conversion produced %d bytes, over the %d byte output limit", len(converted), maxMarkdownOutput)
	}

	trimmed := bytes.TrimSpace(converted)

	if len(trimmed) == 0 {
		return "", nil
	}

	return string(append(trimmed, '\n')), nil
}

// dropInertElements removes markup a reader never sees. The converter drops
// script, style and comments on its own; template and noscript hold content it
// would otherwise emit as if it were part of the page.
func dropInertElements(n *html.Node) {
	child := n.FirstChild

	for child != nil {
		// Held before the node can be detached, or the walk loses its place.
		next := child.NextSibling

		if child.Type == html.ElementNode && inertElements[child.DataAtom] {
			n.RemoveChild(child)
		} else {
			dropInertElements(child)
		}

		child = next
	}
}
