import { describe, expect, beforeEach, it } from "vitest";
import type { RenderResult } from "@testing-library/react";
import { render, fireEvent } from "@testing-library/react";
import { AuthContext } from "~/pages/auth/Auth.context";
import { RootContext } from "~/pages/Root.context";
import mockUser from "~/testing/data/mock_user";
import { subscriptionLink } from "~/utils/billing";
import UpgradeButton from "./UpgradeButton";

interface Props {
  user: User;
  edition?: string;
  text?: string;
  variant?: "contained" | "outlined" | "text";
}

const createWrapper = ({ user, edition, text, variant }: Props): RenderResult =>
  render(
    <RootContext.Provider
      value={{
        mode: "dark",
        setMode: () => {},
        details: { stormkit: { edition: edition as InstanceDetails["stormkit"]["edition"], apiCommit: "", apiVersion: "" } },
      }}
    >
      <AuthContext.Provider value={{ user }}>
        <UpgradeButton text={text} variant={variant} />
      </AuthContext.Provider>
    </RootContext.Provider>,
  );

describe("~/components/UpgradeButton/UpgradeButton", () => {
  let wrapper: RenderResult;

  describe("cloud edition", () => {
    describe("free user", () => {
      let user: User;

      beforeEach(() => {
        user = mockUser({ packageId: "free" });
        wrapper = createWrapper({ user, edition: "cloud" });
      });

      it("should show upgrade text", () => {
        expect(wrapper.getByText("Upgrade to enterprise")).toBeTruthy();
      });

      it("should link to the cloud premium checkout page", () => {
        const link = wrapper.getByRole("link");
        expect(link.getAttribute("href")).toBe(subscriptionLink(user.package.id, user.email, "cloud"));
      });
    });

    describe("paid user", () => {
      let user: User;

      beforeEach(() => {
        user = mockUser({ packageId: "premium" });
        wrapper = createWrapper({ user, edition: "cloud", text: "Manage subscription" });
      });

      it("should show manage subscription text", () => {
        expect(wrapper.getByText("Manage subscription")).toBeTruthy();
      });

      it("should link to the billing portal", () => {
        const link = wrapper.getByRole("link");
        expect(link.getAttribute("href")).toBe(subscriptionLink(user.package.id, user.email, "cloud"));
      });
    });

    describe("text variant", () => {
      it("should render as a link element for free user", () => {
        const user = mockUser({ packageId: "free" });
        wrapper = createWrapper({ user, edition: "cloud", variant: "text" });
        const link = wrapper.getByRole("link");
        expect(link.getAttribute("href")).toBe(subscriptionLink(user.package.id, user.email, "cloud"));
      });

      it("should render as a link element for paid user", () => {
        const user = mockUser({ packageId: "premium" });
        wrapper = createWrapper({ user, edition: "cloud", variant: "text" });
        const link = wrapper.getByRole("link");
        expect(link.getAttribute("href")).toBe(subscriptionLink(user.package.id, user.email, "cloud"));
      });
    });
  });

  describe("self-hosted edition", () => {
    describe("non-admin user", () => {
      let user: User;

      beforeEach(() => {
        user = mockUser({ packageId: "free", isAdmin: false });
        wrapper = createWrapper({ user, edition: "self-hosted" });
      });

      it("should show upgrade text", () => {
        expect(wrapper.getByText("Upgrade to enterprise")).toBeTruthy();
      });

      it("should not render a link", () => {
        expect(wrapper.queryByRole("link")).toBeNull();
      });

      it("should open info modal on click", () => {
        fireEvent.click(wrapper.getByRole("button", { name: /upgrade to enterprise/i }));
        expect(wrapper.getByText("In self-hosted environments, the license is managed by admins.")).toBeTruthy();
      });

      it("should close info modal on close button click", () => {
        fireEvent.click(wrapper.getByRole("button", { name: /upgrade to enterprise/i }));
        fireEvent.click(wrapper.getByRole("button", { name: /close/i }));
        expect(wrapper.queryByText("In self-hosted environments, the license is managed by admins.")).toBeNull();
      });
    });

    describe("admin user", () => {
      let user: User;

      beforeEach(() => {
        user = mockUser({ packageId: "free", isAdmin: true });
        wrapper = createWrapper({ user, edition: "self-hosted" });
      });

      it("should link to the self-hosted checkout page", () => {
        const link = wrapper.getByRole("link");
        expect(link.getAttribute("href")).toBe(subscriptionLink(user.package.id, user.email, "self-hosted"));
      });
    });
  });
});
