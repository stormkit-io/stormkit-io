import { describe, expect, beforeEach, it } from "vitest";
import type { RenderResult } from "@testing-library/react";
import { render } from "@testing-library/react";
import { AuthContext } from "~/pages/auth/Auth.context";
import mockUser from "~/testing/data/mock_user";
import { subscriptionLink } from "~/utils/billing";
import UpgradeButton from "./UpgradeButton";

interface Props {
  user: User;
  text?: string;
  variant?: "contained" | "outlined" | "text";
}

describe("~/components/UpgradeButton/UpgradeButton", () => {
  let wrapper: RenderResult;

  const createWrapper = ({ user, text, variant }: Props) => {
    wrapper = render(
      <AuthContext.Provider value={{ user }}>
        <UpgradeButton text={text} variant={variant} />
      </AuthContext.Provider>,
    );
  };

  describe("free user", () => {
    let user: User;

    beforeEach(() => {
      user = mockUser({ packageId: "free" });
      createWrapper({ user });
    });

    it("should show upgrade text", () => {
      expect(wrapper.getByText("Upgrade to enterprise")).toBeTruthy();
    });

    it("should link to the premium checkout page", () => {
      const link = wrapper.getByRole("link");
      expect(link.getAttribute("href")).toBe(subscriptionLink(user.package.id, user.email));
    });
  });

  describe("paid user", () => {
    let user: User;

    beforeEach(() => {
      user = mockUser({ packageId: "premium" });
      createWrapper({ user, text: "Manage subscription" });
    });

    it("should show manage subscription text", () => {
      expect(wrapper.getByText("Manage subscription")).toBeTruthy();
    });

    it("should link to the billing portal", () => {
      const link = wrapper.getByRole("link");
      expect(link.getAttribute("href")).toBe(subscriptionLink(user.package.id, user.email));
    });
  });

  describe("text variant", () => {
    it("should render as a link element for free user", () => {
      const user = mockUser({ packageId: "free" });
      createWrapper({ user, variant: "text" });
      const link = wrapper.getByRole("link");
      expect(link.getAttribute("href")).toBe(subscriptionLink(user.package.id, user.email));
    });

    it("should render as a link element for paid user", () => {
      const user = mockUser({ packageId: "premium" });
      createWrapper({ user, variant: "text" });
      const link = wrapper.getByRole("link");
      expect(link.getAttribute("href")).toBe(subscriptionLink(user.package.id, user.email));
    });
  });
});
