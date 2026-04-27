import { describe, expect, beforeEach, afterEach, it } from "vitest";
import type { RenderResult } from "@testing-library/react";
import { render } from "@testing-library/react";
import nock from "nock";
import { AuthContext } from "~/pages/auth/Auth.context";
import { RootContext } from "~/pages/Root.context";
import mockUser from "~/testing/data/mock_user";
import UserProfile from "./UserProfile";

interface Props {
  edition: "cloud" | "self-hosted";
  user: User;
  metrics?: UserMetrics;
}

describe("~/pages/user/account/_components/UserProfile", () => {
  let wrapper: RenderResult;

  beforeEach(() => {
    nock.cleanAll();
  });

  afterEach(() => {
    nock.cleanAll();
  });

  const createWrapper = ({ edition, user = mockUser(), metrics }: Props) => {
    wrapper = render(
      <RootContext.Provider
        value={{
          mode: "dark",
          setMode: () => {},
          details: { stormkit: { apiCommit: "", apiVersion: "", edition } },
        }}
      >
        <AuthContext.Provider value={{ user, metrics }}>
          <UserProfile user={user} metrics={metrics} />
        </AuthContext.Provider>
      </RootContext.Provider>,
    );
  };

  describe("cloud edition", () => {
    const metrics: UserMetrics = {
      max: {
        buildMinutes: 300,
        bandwidthInBytes: 1000000,
        storageInBytes: 2000000,
        functionInvocations: 30000,
      },
      used: {
        buildMinutes: 150,
        bandwidthInBytes: 10000,
        storageInBytes: 250000,
        functionInvocations: 30000,
      },
    };

    describe("paid user", () => {
      beforeEach(() => {
        createWrapper({
          edition: "cloud",
          user: mockUser({ packageId: "premium" }),
          metrics,
        });
      });

      it("should display usage metrics", () => {
        expect(wrapper.getByText("Build minutes")).toBeTruthy();
        expect(wrapper.getByText("Bandwidth")).toBeTruthy();
        expect(wrapper.getByText("Function invocations")).toBeTruthy();
        expect(wrapper.getByText("Storage")).toBeTruthy();
        expect(wrapper.getByTestId("Build minutes-usage").textContent).toBe(
          "150 out of 300",
        );
        expect(wrapper.getByTestId("Bandwidth-usage").textContent).toBe(
          "10.0 kB out of 1.0 MB",
        );
        expect(
          wrapper.getByTestId("Function invocations-usage").textContent,
        ).toBe("30,000 out of 30,000");
        expect(wrapper.getByTestId("Storage-usage").textContent).toBe(
          "250.0 kB out of 2.0 MB",
        );
      });

      it("should show manage subscription button", () => {
        expect(wrapper.getByText("Manage subscription")).toBeTruthy();
      });
    });

    describe("free user", () => {
      beforeEach(() => {
        createWrapper({
          edition: "cloud",
          user: mockUser({ packageId: "free" }),
          metrics,
        });
      });

      it("should show upgrade to enterprise button", () => {
        expect(wrapper.getByText("Upgrade to enterprise")).toBeTruthy();
      });
    });
  });

  describe("self-hosted edition", () => {
    beforeEach(() => {
      createWrapper({
        edition: "self-hosted",
        user: mockUser(),
      });
    });

    it("should display self-hosted specific info", () => {
      expect(
        wrapper.getByText(/Visit the admin area to manage your license./),
      ).toBeTruthy();
    });

    it("should not display license section for self-hosted edition", () => {
      expect(() => wrapper.getByText("Self-Hosted License")).toThrow();
    });
  });
});
