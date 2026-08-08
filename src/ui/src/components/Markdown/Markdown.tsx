import type { SxProps } from "@mui/material";
import { useMemo } from "react";
import createDOMPurify from "dompurify";
import { marked } from "marked";
import Typography from "@mui/material/Typography";
import {
  highlightToHtml,
  tokenStyles,
  type Language,
} from "~/utils/helpers/highlight";

interface Props {
  children: string;
  /** See SafeHtmlOptions.allowImages. */
  allowImages?: boolean;
  sx?: SxProps;
}

interface SafeHtmlOptions {
  /**
   * Permits `img`. Off by default: an image is fetched the moment the content
   * renders, so it reports the reader's IP to whatever host the author picked,
   * without the reader choosing to go there the way a link requires. Worth it
   * for first-party content like the changelog, not for text any teammate can
   * store on a record.
   */
  allowImages?: boolean;
}

const LANGUAGE_CLASS = /^language-[a-z0-9-]+$/i;

// An href or src is either an http(s)/mailto URL or a same-document reference
// starting with / or #. `/(?!/)` and not `/` alone, so a protocol-relative
// `//evil.test` cannot pass itself off as an in-app path. Spelled out as an
// allowlist rather than relaxed to DOMPurify's default, which is where a
// bypass would creep back in.
const SAFE_URI = /^(?:https?:|mailto:|#|\/(?!\/))/i;

// DOMPurify strips these before its own URI check, so anything comparing a URI
// by hand has to strip them too — `da&#9;ta:` is `data:` to the browser.
const URI_WHITESPACE =
  /[\u0000-\u0020\u00A0\u1680\u180E\u2000-\u2029\u205F\u3000]/g;

function isSafeURI(value: string): boolean {
  return SAFE_URI.test(value.replace(URI_WHITESPACE, ""));
}

// Inline formatting and structure only. Anything that can execute, embed or
// phone home (script, iframe, style, form, event handlers) is dropped rather
// than escaped, because the source is user-authored text stored verbatim.
//
// svg, math, style, noscript and template are deliberately absent: they are the
// namespace-confusion primitives every known mXSS mutation needs, and
// highlightCodeBlocks below re-parses the sanitized output, so sanitizing is
// not the last step. Do not add them without moving the highlighting pass
// before the final sanitize.
const ALLOWED_TAGS = [
  "h1",
  "h2",
  "h3",
  "h4",
  "h5",
  "h6",
  "p",
  "a",
  "ul",
  "ol",
  "li",
  "strong",
  "em",
  "b",
  "i",
  "code",
  "pre",
  "blockquote",
  "table",
  "thead",
  "tbody",
  "tr",
  "th",
  "td",
  "br",
  "hr",
];

const ALLOWED_ATTR = ["href", "title", "target", "rel", "class"];

// Own instance rather than the shared default export: the hook below is
// registered globally on whichever instance it is attached to, and other
// callers sanitize their own content with different rules.
const purify = createDOMPurify(window);

/**
 * Rewrites the body of each fenced code block whose language we can parse into
 * class-tagged spans.
 *
 * Runs on already-sanitized HTML, and highlights the block's textContent rather
 * than its markup, so the spans it introduces cannot carry anything the
 * sanitizer just removed.
 */
function highlightCodeBlocks(html: string): string {
  if (!html.includes("<code")) {
    return html;
  }

  const doc = new DOMParser().parseFromString(html, "text/html");
  const blocks = doc.querySelectorAll("pre > code[class]");

  if (blocks.length === 0) {
    return html;
  }

  blocks.forEach(block => {
    const language = block.className
      .replace(/^language-/i, "")
      .toLowerCase() as Language;

    const highlighted = highlightToHtml(block.textContent || "", language);

    if (highlighted !== null) {
      block.innerHTML = highlighted;
    }
  });

  return doc.body.innerHTML;
}

purify.addHook("afterSanitizeAttributes", node => {
  // ALLOWED_URI_REGEXP is not the last word on its own: DOMPurify carves out
  // data: URIs for img and the other media tags regardless of it, which is how
  // a `data:image/svg+xml,<svg onload=...>` source survives the config. Re-check
  // both attributes here so the allowlist is actually the allowlist.
  ["href", "src"].forEach(attr => {
    const value = node.getAttribute?.(attr);

    if (value !== null && value !== undefined && !isSafeURI(value)) {
      node.removeAttribute(attr);
    }
  });

  // Links off the app are authored by one team member and clicked by another,
  // so they open in a new tab and never hand the opener over to the
  // destination. In-app paths and anchors navigate in place.
  if (node.tagName === "A" && node.hasAttribute("href")) {
    if (/^https?:/i.test(node.getAttribute("href") || "")) {
      node.setAttribute("target", "_blank");
      node.setAttribute("rel", "noopener noreferrer nofollow");
    }
  }

  // `class` is allowed only so fenced code blocks keep the language marker
  // marked emits. Anything else is dropped rather than letting authored text
  // borrow arbitrary styles from the app.
  if (node.hasAttribute?.("class")) {
    const className = node.getAttribute("class") || "";

    if (LANGUAGE_CLASS.test(className)) {
      node.setAttribute("class", className.trim());
    } else {
      node.removeAttribute("class");
    }
  }
});

export function toSafeHtml(
  markdown: string,
  { allowImages }: SafeHtmlOptions = {}
): string {
  const html = purify.sanitize(marked.parse(markdown || "") as string, {
    ALLOWED_TAGS: allowImages ? [...ALLOWED_TAGS, "img"] : ALLOWED_TAGS,
    ALLOWED_ATTR: allowImages ? [...ALLOWED_ATTR, "src", "alt"] : ALLOWED_ATTR,
    ALLOW_DATA_ATTR: false,
    // Backed up by the re-check in the hook above, which covers the data: URI
    // exception this option does not.
    ALLOWED_URI_REGEXP: SAFE_URI,
  });

  return highlightCodeBlocks(html);
}

/**
 * Renders a markdown string as sanitized HTML, inheriting the surrounding
 * typography so it reads as body text rather than raw HTML defaults.
 */
export default function Markdown({ children, allowImages, sx }: Props) {
  const html = useMemo(
    () => toSafeHtml(children, { allowImages }),
    [children, allowImages]
  );

  if (!html) {
    return null;
  }

  return (
    <Typography
      component="div"
      data-testid="markdown"
      dangerouslySetInnerHTML={{ __html: html }}
      sx={{
        wordBreak: "break-word",
        "& > *:first-of-type": { mt: 0 },
        "& > *:last-child": { mb: 0 },
        "& p, & ul, & ol, & blockquote, & pre, & table": { my: 1 },
        "& ul, & ol": { pl: 3 },
        "& h1, & h2, & h3, & h4, & h5, & h6": {
          my: 1,
          fontSize: "inherit",
          fontWeight: "bold",
        },
        "& a": { color: "text.primary" },
        "& code": {
          bgcolor: "container.transparent",
          borderRadius: 1,
          px: 0.5,
        },
        "& pre": {
          bgcolor: "container.transparent",
          borderRadius: 1,
          p: 1,
          overflowX: "auto",
          "& code": { bgcolor: "transparent", px: 0 },
          ...tokenStyles,
        },
        "& blockquote": {
          borderLeft: "2px solid",
          borderColor: "container.transparent",
          pl: 1.5,
          ml: 0,
        },
        "& table": { borderCollapse: "collapse" },
        "& th, & td": {
          border: "1px solid",
          borderColor: "container.border",
          px: 1,
          py: 0.5,
        },
        ...sx,
      }}
    />
  );
}
