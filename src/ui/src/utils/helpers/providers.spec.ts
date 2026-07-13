import { getLogoForProvider, parseRepo } from "./providers";

describe("~/utils/helpers/providers", () => {
  test("parseRepo: parses the given repository as expected", () => {
    expect(parseRepo("github/stormkit-io/app-stormkit-io")).toEqual({
      provider: "github",
      repo: "stormkit-io/app-stormkit-io",
    });

    expect(parseRepo("bitbucket/stormkit-io/app-stormkit-io")).toEqual({
      provider: "bitbucket",
      repo: "stormkit-io/app-stormkit-io",
    });

    expect(parseRepo("gitlab/stormkit-io/app-stormkit-io")).toEqual({
      provider: "gitlab",
      repo: "stormkit-io/app-stormkit-io",
    });
  });

  test("getLogoForProvider: returns the correct logo for the provider", () => {
    expect(getLogoForProvider("github")).toEqual("test-file-stub");
    expect(getLogoForProvider("bitbucket")).toEqual("test-file-stub");
    expect(getLogoForProvider("gitlab")).toEqual("test-file-stub");
  });
});
