import { describe, expect, it } from "vitest";
import { detectLanguage, highlightToHtml } from "./highlight";

describe("~/utils/helpers/highlight.ts", () => {
  describe("detectLanguage", () => {
    it("detects json objects and arrays", () => {
      expect(detectLanguage('{"a":1}')).toBe("json");
      expect(detectLanguage("[1, 2, 3]")).toBe("json");
      expect(detectLanguage('\n  {"a":1}\n')).toBe("json");
    });

    it("detects html documents and fragments", () => {
      expect(detectLanguage("<!DOCTYPE html>\n<html></html>")).toBe("html");
      expect(detectLanguage("<div>hi</div>")).toBe("html");
      expect(detectLanguage('<a href="/x">hi</a>')).toBe("html");
    });

    it("returns undefined for plain text and broken json", () => {
      expect(detectLanguage("")).toBeUndefined();
      expect(detectLanguage("   ")).toBeUndefined();
      expect(detectLanguage("No payload")).toBeUndefined();
      expect(detectLanguage("{not json")).toBeUndefined();
      expect(detectLanguage("5 < 6 and 7 > 2")).toBeUndefined();
    });
  });

  describe("highlightToHtml", () => {
    it("returns null when the language is unknown", () => {
      expect(highlightToHtml("print(1)", undefined)).toBe(null);
    });

    // The language comes from a fence's info string, so inherited keys must not
    // resolve to a parser.
    it("returns null for inherited object keys", () => {
      const inherited = ["constructor", "toString", "valueOf", "__proto__"];

      inherited.forEach(language => {
        expect(highlightToHtml("x", language as never)).toBe(null);
      });
    });

    it("tags json tokens", () => {
      const html = highlightToHtml('{"a": 1, "b": true}', "json")!;

      expect(html).toContain('<span class="tok-propertyName">&quot;a&quot;');
      expect(html).toContain('<span class="tok-number">1</span>');
      expect(html).toContain('<span class="tok-bool">true</span>');
    });

    it("tags html tokens and escapes the markup", () => {
      const html = highlightToHtml('<a href="/x">hi</a>', "html")!;

      expect(html).toContain('<span class="tok-typeName">a</span>');
      expect(html).toContain('<span class="tok-propertyName">href</span>');
      expect(html).toContain("&lt;");
      expect(html).not.toContain('<a href="/x"');
    });

    it("escapes content that would otherwise become markup", () => {
      const html = highlightToHtml(
        '{"a": "<img src=x onerror=alert(1)>"}',
        "json"
      )!;

      expect(html).not.toContain("<img");
      expect(html).toContain("&lt;img");
    });
  });
});
