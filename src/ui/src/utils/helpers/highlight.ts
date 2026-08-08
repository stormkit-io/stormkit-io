import type { Parser } from "@lezer/common";
import { classHighlighter, highlightCode } from "@lezer/highlight";
import { jsonLanguage } from "@codemirror/lang-json";
import { htmlLanguage } from "@codemirror/lang-html";

export type Language = "json" | "html";

const parsers: Record<Language, Parser> = {
  json: jsonLanguage.parser,
  html: htmlLanguage.parser,
};

/**
 * Token colors for highlighted output, keyed by the classes
 * @lezer/highlight's classHighlighter emits.
 *
 * JSON keys and HTML attribute names both arrive as propertyName, and HTML tag
 * names as typeName. Colors are palette entries so highlighting follows the
 * active theme rather than assuming a dark background.
 */
export const tokenStyles = {
  "& .tok-propertyName": { color: "info.main" },
  "& .tok-string": { color: "success.main" },
  "& .tok-number, & .tok-bool, & .tok-keyword, & .tok-atom": {
    color: "warning.main",
  },
  "& .tok-typeName": { color: "error.main" },
  "& .tok-punctuation, & .tok-operator": { color: "text.secondary" },
  "& .tok-comment": { color: "text.secondary", fontStyle: "italic" },
};

export function escapeHtml(text: string): string {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

/**
 * Guesses the language of a request or response body, which arrives without a
 * content type attached. Only the two languages we can highlight are detected;
 * anything else is left to render as plain text.
 */
export function detectLanguage(code: string): Language | undefined {
  const trimmed = code.trim();

  if (!trimmed) {
    return undefined;
  }

  if (/^[[{]/.test(trimmed)) {
    try {
      JSON.parse(trimmed);
      return "json";
    } catch {
      return undefined;
    }
  }

  if (/^<(?:!doctype|\?xml|\/?[a-z][a-z0-9-]*)[\s>/]/i.test(trimmed)) {
    return "html";
  }

  return undefined;
}

/**
 * Renders code as class-tagged spans.
 *
 * Every span is built from the plain text passed in, with the text escaped, so
 * the markup this produces never carries anything from the source beyond its
 * characters. Returns null when the language is unknown, leaving the caller to
 * render the text as is.
 *
 * The language reaches here from a fenced block's info string, so it is an
 * arbitrary word rather than a `Language`: it is looked up as an own key, and
 * parsing is guarded, because this runs during render and a throw here would
 * take the page down.
 */
export function highlightToHtml(
  code: string,
  language?: Language
): string | null {
  if (!language || !Object.hasOwn(parsers, language)) {
    return null;
  }

  const parser = parsers[language];

  let html = "";

  try {
    highlightCode(
      code,
      parser.parse(code),
      classHighlighter,
      (text, classes) => {
        html += classes
          ? `<span class="${classes}">${escapeHtml(text)}</span>`
          : escapeHtml(text);
      },
      () => {
        html += "\n";
      }
    );
  } catch {
    return null;
  }

  return html;
}
