import { describe, it, expect } from "vitest";
import { formatRepo, toRepoAddr } from "./helpers";

describe("~/pages/apps/[id]/settings/helpers.tsx", () => {
  const forgejoUrl = "https://git.example.org";

  it("toRepoAddr: returns the clone address of a Forgejo repository", () => {
    expect(toRepoAddr("forgejo/stormkit/app", forgejoUrl)).toBe(
      "https://git.example.org/stormkit/app.git"
    );
  });

  it("toRepoAddr: returns an empty address without the instance URL", () => {
    expect(toRepoAddr("forgejo/stormkit/app")).toBe("");
  });

  it("formatRepo: converts a Forgejo address to the Stormkit format", () => {
    expect(formatRepo("https://git.example.org/stormkit/app.git", forgejoUrl)).toBe(
      "forgejo/stormkit/app"
    );
  });

  it("formatRepo: keeps handling the hosted providers", () => {
    expect(formatRepo("https://github.com/stormkit/app.git", forgejoUrl)).toBe(
      "github/stormkit/app"
    );
  });
});
