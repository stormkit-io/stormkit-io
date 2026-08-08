import { describe, expect, it } from "vitest";
import { render } from "@testing-library/react";
import Markdown, { toSafeHtml } from "./Markdown";

describe("~/components/Markdown/Markdown.tsx", () => {
  it("renders markdown formatting", () => {
    const wrapper = render(
      <Markdown>{"# Title\n\nSome **bold** text and `code`."}</Markdown>
    );

    const html = wrapper.getByTestId("markdown").innerHTML;

    expect(html).toContain("<h1>Title</h1>");
    expect(html).toContain("<strong>bold</strong>");
    expect(html).toContain("<code>code</code>");
  });

  it("renders nothing when the content is empty", () => {
    const wrapper = render(<Markdown>{""}</Markdown>);

    expect(wrapper.queryByTestId("markdown")).toBe(null);
  });

  it("strips scripts and event handlers", () => {
    const html = toSafeHtml(
      '<script>alert(1)</script><img src=x onerror="alert(1)"><p onclick="alert(1)">hi</p>'
    );

    expect(html).not.toContain("script");
    expect(html).not.toContain("onerror");
    expect(html).not.toContain("onclick");
    expect(html).toContain("hi");
  });

  it("strips javascript: and data: links but keeps regular ones", () => {
    expect(toSafeHtml("[click](javascript:alert(1))")).not.toContain(
      "javascript:"
    );

    expect(
      toSafeHtml('<a href="data:text/html,<script>alert(1)</script>">x</a>')
    ).not.toContain("data:");

    expect(toSafeHtml("[docs](https://www.stormkit.io)")).toContain(
      'href="https://www.stormkit.io"'
    );
  });

  it("opens links in a new tab without handing over the opener", () => {
    const html = toSafeHtml("[docs](https://www.stormkit.io)");

    expect(html).toContain('target="_blank"');
    expect(html).toContain('rel="noopener noreferrer nofollow"');
  });

  it("keeps in-app paths and anchors, navigating them in place", () => {
    const path = toSafeHtml("[runbook](/apps/12/deployments)");

    expect(path).toContain('href="/apps/12/deployments"');
    expect(path).not.toContain("target=");

    const anchor = toSafeHtml("[jump](#section)");

    expect(anchor).toContain('href="#section"');
    expect(anchor).not.toContain("target=");
  });

  // A protocol-relative URL leaves the app while looking like a path.
  it("strips protocol-relative links", () => {
    expect(toSafeHtml("[x](//evil.test/p)")).not.toContain("evil.test");
  });

  it("does not execute injected script when rendered", () => {
    const fired: string[] = [];
    (window as any).__xss = () => fired.push("fired");

    const wrapper = render(
      <Markdown>
        {'<script>window.__xss()</script>' +
          '<img src=x onerror="window.__xss()">' +
          '<svg><animate onbegin="window.__xss()" /></svg>' +
          "\n\nsafe text"}
      </Markdown>
    );

    expect(wrapper.getByTestId("markdown").querySelector("script")).toBe(null);
    expect(wrapper.getByTestId("markdown").innerHTML).not.toContain("__xss");
    expect(fired).toEqual([]);

    delete (window as any).__xss;
  });

  it("highlights json code fences", () => {
    const html = toSafeHtml('```json\n{"a": 1, "b": true}\n```');

    expect(html).toContain('<span class="tok-propertyName">"a"</span>');
    expect(html).toContain('<span class="tok-number">1</span>');
    expect(html).toContain('<span class="tok-bool">true</span>');
  });

  it("highlights html code fences", () => {
    const html = toSafeHtml('```html\n<a href="/x">hi</a>\n```');

    expect(html).toContain('<span class="tok-typeName">a</span>');
    expect(html).toContain('<span class="tok-propertyName">href</span>');
    // The fenced markup stays escaped rather than becoming a real element.
    expect(html).toContain("&lt;");
    expect(html).not.toContain('<a href="/x"');
  });

  it("leaves fences in unsupported languages alone", () => {
    const html = toSafeHtml("```python\nprint(1)\n```");

    expect(html).not.toContain("tok-");
    expect(html).toContain("print(1)");
  });

  // A fence named after an inherited object key used to resolve to a truthy
  // parser and throw mid-render, blanking the page.
  it("renders a fence named after an object property", () => {
    const wrapper = render(
      <Markdown>{"Docs\n\n```constructor\nhello\n```"}</Markdown>
    );

    const html = wrapper.getByTestId("markdown").innerHTML;

    expect(html).toContain("hello");
    expect(html).not.toContain("tok-");
  });

  it("keeps only language- classes on code blocks", () => {
    const html = toSafeHtml(
      '<p class="MuiButton-root">x</p><pre><code class="language-json">1</code></pre>'
    );

    expect(html).not.toContain("MuiButton-root");
    expect(html).toContain('class="language-json"');
  });

  it("drops images by default and keeps them when allowed", () => {
    expect(toSafeHtml("![x](https://www.stormkit.io/a.png)")).not.toContain(
      "<img"
    );

    const allowed = toSafeHtml("![x](https://www.stormkit.io/a.png)", {
      allowImages: true,
    });

    expect(allowed).toContain('src="https://www.stormkit.io/a.png"');

    // The URI allowlist still applies, so no data:/javascript: sources.
    expect(
      toSafeHtml('<img src="data:image/svg+xml,<svg/onload=alert(1)>">', {
        allowImages: true,
      })
    ).not.toContain("data:");
  });

  it("drops embedding and styling tags", () => {
    const html = toSafeHtml(
      '<iframe src="https://evil.test"></iframe>' +
        "<style>body{display:none}</style>" +
        '<form action="https://evil.test"><input name="pw"></form>' +
        "<p>kept</p>"
    );

    expect(html).not.toContain("iframe");
    expect(html).not.toContain("<style");
    expect(html).not.toContain("<form");
    expect(html).not.toContain("<input");
    expect(html).toContain("<p>kept</p>");
  });
});
