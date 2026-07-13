import type { Scope } from "nock";
import { describe, expect, it } from "vitest";
import { RenderResult, waitFor } from "@testing-library/react";
import { fireEvent, render } from "@testing-library/react";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import {
  mockFetchAPIKeys,
  mockDeleteAPIKey,
} from "~/testing/nocks/nock_api_keys";
import TabAPIKey from "./TabAPIKey";

interface WrapperProps {
  app?: App;
  environments?: Environment[];
  hash?: string;
  setRefreshToken?: () => void;
}

describe("~/pages/apps/[id]/environments/[env-id]/config/_components/TabAPIKey.tsx", () => {
  let wrapper: RenderResult;
  let fetchScope: Scope;
  let currentApp: App;
  let currentEnv: Environment;
  let currentEnvs: Environment[];

  const createWrapper = ({ app, environments }: WrapperProps) => {
    currentApp = app || mockApp();
    currentApp.id = "1"; // Same as api key id

    currentEnvs = environments || mockEnvironments({ app: currentApp });
    currentEnv = currentEnvs[0];

    fetchScope = mockFetchAPIKeys({
      appId: currentApp.id,
      envId: currentEnv.id!,
    });

    wrapper = render(<TabAPIKey app={currentApp} environment={currentEnv} />);
  };

  it("should fetch api keys", async () => {
    createWrapper({});

    await waitFor(() => {
      expect(fetchScope.isDone()).toBe(true);

      const subheader =
        "This key will allow you to interact with our API and modify this environment.";

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
      appId: currentApp.id,
      envId: currentEnv.id!,
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
