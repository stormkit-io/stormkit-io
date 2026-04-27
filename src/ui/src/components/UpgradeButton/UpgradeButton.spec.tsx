import { describe, expect, beforeEach, it } from "vitest";
import type { RenderResult } from "@testing-library/react";
import { render } from "@testing-library/react";
import { AuthContext } from "~/pages/auth/Auth.context";
import mockUser from "~/testing/data/mock_user";
import { portalLink, paymentLink } from "~/utils/billing";
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
    beforeEach(() => {
      createWrapper({ user: mockUser({ packageId: "free" }) });
    });

    it("should show upgrade text", () => {
      expect(wrapper.getByText("Upgrade to enterprise")).toBeTruthy();
    });

    it("should link to the premium payment page", () => {
      const link = wrapper.getByRole("link");
      expect(link.getAttribute("href")).toBe(paymentLink.premium);
    });
  });

  describe("paid user", () => {
    beforeEach(() => {
      createWrapper({
        user: mockUser({ packageId: "premium" }),
        text: "Manage subscription",
      });
    });

    it("should show manage subscription text", () => {
      expect(wrapper.getByText("Manage subscription")).toBeTruthy();
    });

    it("should link to the billing portal", () => {
      const link = wrapper.getByRole("link");
      expect(link.getAttribute("href")).toBe(portalLink);
    });
  });

  describe("text variant", () => {
    it("should render as a link element for free user", () => {
      createWrapper({ user: mockUser({ packageId: "free" }), variant: "text" });
      const link = wrapper.getByRole("link");
      expect(link.getAttribute("href")).toBe(paymentLink.premium);
    });

    it("should render as a link element for paid user", () => {
      createWrapper({
        user: mockUser({ packageId: "premium" }),
        variant: "text",
      });
      const link = wrapper.getByRole("link");
      expect(link.getAttribute("href")).toBe(portalLink);
    });
  });
});
