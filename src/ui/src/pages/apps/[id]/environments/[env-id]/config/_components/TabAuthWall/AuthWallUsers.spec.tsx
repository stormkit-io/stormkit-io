import type { Scope } from "nock";
import type { RenderResult } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, waitFor, fireEvent } from "@testing-library/react";
import mockApp from "~/testing/data/mock_app";
import mockEnvironment from "~/testing/data/mock_environment";
import {
  mockFetchLogins,
  mockDeleteLogins,
} from "~/testing/nocks/nock_auth_wall";
import AuthWallUsers from "./AuthWallUsers";

vi.mock("~/utils/helpers/date", () => ({
  formatDate: (t?: number) => (t ? "21.09.2022 - 21:30" : ""),
}));

describe("~/pages/apps/[id]/environments/[env-id]/config/_components/TabAuthWall/AuthWallUsers.tsx", () => {
  let wrapper: RenderResult;
  let app: App;
  let env: Environment;
  let scope: Scope;

  beforeEach(() => {
    app = mockApp();
    env = mockEnvironment({ app });
  });

  const createWrapper = () => {
    wrapper = render(<AuthWallUsers app={app} environment={env} />);
  };

  describe("with empty list", () => {
    beforeEach(() => {
      scope = mockFetchLogins({
        appId: app.id,
        envId: env.id!,
        response: { logins: [] },
      });

      createWrapper();
    });

    it("should display an empty list", async () => {
      expect(wrapper.getByTestId("card-loading")).toBeTruthy();

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
      });

      expect(() => wrapper.getByTestId("card-loading")).toThrow();
      expect(
        wrapper.getByText(
          "There are no logins for this environment. Click the button above to add a new user."
        )
      ).toBeTruthy();

      // Should not contain header row
      expect(() => wrapper.getByText("Email")).toThrow();
      expect(() => wrapper.getByText("Last login")).toThrow();

      // Should not contain remove button
      expect(() => wrapper.getByText("Remove selected")).toThrow();
    });
  });

  describe("with pre-existing users", () => {
    beforeEach(() => {
      scope = mockFetchLogins({
        appId: app.id,
        envId: env.id!,
        response: {
          logins: [
            { email: "email-1@example.org", lastLogin: 0, id: "1" },
            {
              email: "email-2@example.org",
              lastLogin: 1743904881,
              id: "2",
            },
          ],
        },
      });

      createWrapper();
    });

    it("should display a list of users", async () => {
      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
      });

      expect(() =>
        wrapper.getByText(
          "There are no logins for this environment. Click the button above to add a new user."
        )
      ).toThrow();

      await waitFor(() => {
        expect(wrapper.getByText("Email")).toBeTruthy();
        expect(wrapper.getByText("Last login")).toBeTruthy();
        expect(wrapper.getByText("email-1@example.org")).toBeTruthy();
        expect(wrapper.getByText("email-2@example.org")).toBeTruthy();
        expect(wrapper.getByText("21.09.2022 - 21:30")).toBeTruthy();
      });
    });

    it("should remove selected users", async () => {
      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
      });

      const button = wrapper.getByText("Remove selected");

      // The remove selected should be disabled initially
      expect(button.getAttribute("disabled")).toBe("");

      await waitFor(() => {
        expect(wrapper.getByLabelText("email-1@example.org")).toBeTruthy();
      });

      fireEvent.click(wrapper.getByLabelText("email-1@example.org"));

      // Then it should be not disabled anymore
      expect(button.getAttribute("disabled")).toBe(null);

      fireEvent.click(button);

      expect(wrapper.getByText("Confirm action")).toBeTruthy();
      expect(wrapper.getByText(/about to remove 1 login./)).toBeTruthy();

      const deleteScope = mockDeleteLogins({
        appId: app.id,
        envId: env.id!,
        loginIds: "1",
      });

      const newFetchScope = mockFetchLogins({
        appId: app.id,
        envId: env.id!,
        response: { logins: [] },
      });

      fireEvent.click(wrapper.getByText("Yes, continue"));

      await waitFor(() => {
        expect(deleteScope.isDone()).toBe(true);
        expect(newFetchScope.isDone()).toBe(true);
      });
    });
  });
});
