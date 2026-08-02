import { describe, expect, it } from "vitest";
import { formatBody } from "./prettyPrint";

describe("~/apps/[id]/environments/[env-id]/function-triggers/_components/prettyPrint.ts", () => {
  it("indents valid json", () => {
    expect(formatBody(`{"hello":"world"}`)).toBe(`{\n  "hello": "world"\n}`);
  });

  it("returns non-json content as is", () => {
    expect(formatBody("<html><body>hi</body></html>")).toBe(
      "<html><body>hi</body></html>"
    );
  });

  it("returns the fallback for empty values", () => {
    expect(formatBody(undefined, "No payload")).toBe("No payload");
    expect(formatBody("", "No payload")).toBe("No payload");
  });

  it("defaults the fallback to an empty string", () => {
    expect(formatBody(undefined)).toBe("");
  });
});
