import { isSelfHosted } from "./instance";

describe("utils/helpers/instance", () => {
  test("isSelfHosted", () => {
    [
      "https://www.stormkit.io",
      "https://www.stormkit.dev",
      "https://www.stormkit.io/auth",
      "https://www.stormkit.dev/auth",
      "https://stormkit.dev/auth?my=1",
      "https://stormkit.io/auth?my=1",
      "https://stormkit.dev",
      "https://stormkit.io",
    ].forEach(url => {
      expect(isSelfHosted(url)).toBe(false);
    });

    [
      "https://www.example.io/auth",
      "https://www.example.dev/auth",
      "https://www.example.io/auth?my=1",
      "https://www.example.dev/auth?my=1",
      "https://example.dev/auth",
      "https://example.io/auth",
      "https://example.dev",
      "https://example.io",
    ].forEach(url => {
      expect(isSelfHosted(url)).toBe(true);
    });
  });
});
