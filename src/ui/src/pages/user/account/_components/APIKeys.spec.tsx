import type { Scope } from "nock";
import { describe, expect, it } from "vitest";
import { RenderResult, waitFor } from "@testing-library/react";
import { fireEvent, render } from "@testing-library/react";
import {
  mockFetchAPIKeys,
  mockDeleteAPIKey,
} from "~/testing/nocks/nock_api_keys";
import APIKeys from "./APIKeys";
import { mockUser } from "~/testing/data";

interface WrapperProps {
  user?: User;
  setRefreshToken?: () => void;
}

describe("~/pages/user/account/_components/APIKeys.tsx", () => {
  let wrapper: RenderResult;
  let fetchScope: Scope;
  let currentUser: User;

  const createWrapper = ({ user }: WrapperProps) => {
    currentUser = user || mockUser();

    fetchScope = mockFetchAPIKeys({
      userId: currentUser.id,
    });

    wrapper = render(<APIKeys user={currentUser} />);
  };

  it("should fetch api keys", async () => {
    createWrapper({});

    await waitFor(() => {
      expect(fetchScope.isDone()).toBe(true);

      const subheader =
        "This key will grant you programmatic access to everything in your Stormkit account.";

      // Header
      expect(wrapper.getByText("API Keys")).toBeTruthy();
      expect(wrapper.getByText(subheader)).toBeTruthy();

      // API Keys
      expect(wrapper.getByText("Default")).toBeTruthy();
      expect(wrapper.getByText("CI")).toBeTruthy();
    });
  });

  it("should delete api key", async () => {
    createWrapper({});

    await waitFor(() => {
      expect(wrapper.getByText("CI")).toBeTruthy();
    });

    fireEvent.click(wrapper.getByLabelText("expand-9868814106"));
    fireEvent.click(wrapper.getByText("Delete"));

    await waitFor(() => {
      expect(wrapper.getByText("Confirm action")).toBeTruthy();
    });

    const scope = mockDeleteAPIKey({
      keyId: "9868814106",
    });

    // Should refetch api keys
    const fetchScope2 = mockFetchAPIKeys({
      userId: currentUser.id,
      response: { keys: [] },
    });

    fireEvent.click(wrapper.getByText("Yes, continue"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(fetchScope2.isDone()).toBe(true);
      // Should close modal
      expect(() => wrapper.getByText("Confirm action")).toThrow();
      // Should no longer have API keys
      expect(() => wrapper.getByText("Default")).toThrow();
    });
  });
});
