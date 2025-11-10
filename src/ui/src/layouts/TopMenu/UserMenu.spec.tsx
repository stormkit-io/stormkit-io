import { render, RenderResult } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { mockUser } from "~/testing/data";
import UserMenu from "./UserMenu";

vi.mock("~/utils/storage", () => ({
  LocalStorage: { get: () => "github", set: () => "" },
}));

describe("~/components/UserMenu.tsx", () => {
  let wrapper: RenderResult;

  const createWrapper = ({ user }: { user?: User } = {}) => {
    wrapper = render(<UserMenu user={user || mockUser()} />);
  };

  describe("with paid tier user", () => {
    beforeEach(() => {
      createWrapper();
    });

    it.each`
      text               | href
      ${"Documentation"} | ${"https://www.stormkit.io/docs/welcome/getting-started"}
      ${"Account"}       | ${"/user/account"}
      ${"Log out"}       | ${"/logout"}
    `("should load menu item: $text", ({ text, href }) => {
      expect(wrapper.getByText(text).getAttribute("href")).toBe(href);
    });

    it.each`
      label        | href
      ${"X"}       | ${"https://twitter.com/stormkitio"}
      ${"YouTube"} | ${"https://www.youtube.com/channel/UC6C_-UuAiIWlGOIokT03lRQ"}
      ${"Discord"} | ${"https://discord.gg/6yQWhyY"}
      ${"Email"}   | ${"mailto:hello@stormkit.io"}
    `("should load menu item: $text", ({ label, href }) => {
      expect(wrapper.getByLabelText(label).getAttribute("href")).toBe(href);
    });
  });
});
