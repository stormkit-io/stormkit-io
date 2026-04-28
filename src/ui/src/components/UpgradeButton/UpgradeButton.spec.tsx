import { describe, expect, beforeEach, it } from "vitest";
import type { RenderResult } from "@testing-library/react";
import { render } from "@testing-library/react";
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

  describe("cloud free user", () => {
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

  describe("self-hosted free user", () => {
    let user: User;

    beforeEach(() => {
      user = mockUser({ packageId: "free" });
      wrapper = createWrapper({ user, edition: "self-hosted" });
    });

    it("should link to the self-hosted premium checkout page", () => {
      const link = wrapper.getByRole("link");
      expect(link.getAttribute("href")).toBe(subscriptionLink(user.package.id, user.email, "self-hosted"));
    });
  });

  describe("paid user", () => {
    let user: User;

    beforeEach(() => {
      user = mockUser({ packageId: "premium" });
      wrapper = createWrapper({ user, text: "Manage subscription" });
    });

    it("should show manage subscription text", () => {
      expect(wrapper.getByText("Manage subscription")).toBeTruthy();
    });

    it("should link to the billing portal", () => {
      const link = wrapper.getByRole("link");
      expect(link.getAttribute("href")).toBe(subscriptionLink(user.package.id, user.email, undefined));
    });
  });

  describe("text variant", () => {
    it("should render as a link element for free user", () => {
      const user = mockUser({ packageId: "free" });
      wrapper = createWrapper({ user, variant: "text" });
      const link = wrapper.getByRole("link");
      expect(link.getAttribute("href")).toBe(subscriptionLink(user.package.id, user.email, undefined));
    });

    it("should render as a link element for paid user", () => {
      const user = mockUser({ packageId: "premium" });
      wrapper = createWrapper({ user, variant: "text" });
      const link = wrapper.getByRole("link");
      expect(link.getAttribute("href")).toBe(subscriptionLink(user.package.id, user.email, undefined));
    });
  });
});
