import type { RenderResult } from "@testing-library/react";
import { describe, it, expect, beforeEach } from "vitest";
import { render, waitFor, fireEvent } from "@testing-library/react";
import mockApp from "~/testing/data/mock_app";
import mockEnvironment from "~/testing/data/mock_environment";
import * as actions from "~/testing/nocks/nock_auth_wall";
import AuthWallConfig from "./AuthWallConfig";

const { mockFetchAuthWallConfig, mockUpdateAuthWallConfig } = actions;

describe("~/pages/apps/[id]/environments/[env-id]/config/_components/TabAuthWall/AuthWallConfig.tsx", () => {
  let wrapper: RenderResult;
  let app: App;
  let env: Environment;

  beforeEach(() => {
    app = mockApp();
    env = mockEnvironment({ app });
  });

  const createWrapper = () => {
    wrapper = render(<AuthWallConfig app={app} environment={env} />);
  };

  it.each`
    config   | selectedItem                                      | expectedStatus
    ${""}    | ${"Auth Wall is disabled"}                        | ${"Disabled"}
    ${"all"} | ${"Protect all endpoints including your domains"} | ${"Enabled"}
    ${"dev"} | ${"Protect only deployment previews"}             | ${"Enabled"}
  `(
    "select the correct dropdown item based on the API response",
    async ({ config, selectedItem, expectedStatus }) => {
      const scope = mockFetchAuthWallConfig({
        appId: app.id,
        envId: env.id!,
        response: { authwall: config },
      });

      createWrapper();
      expect(wrapper.getByTestId("card-loading")).toBeTruthy();

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(() => wrapper.getByTestId("card-loading")).toThrow();
        expect(wrapper.getByText(selectedItem)).toBeTruthy();
        expect(wrapper.getByText(expectedStatus)).toBeTruthy();
      });
    }
  );

  it("select the correct dropdown item based on the API response", async () => {
    const fetchScope = mockFetchAuthWallConfig({
      appId: app.id,
      envId: env.id!,
      response: { authwall: "dev" },
    });

    createWrapper();

    await waitFor(() => {
      expect(fetchScope.isDone()).toBe(true);
    });

    const updateScope = mockUpdateAuthWallConfig({
      appId: app.id,
      envId: env.id!,
      authwall: "dev",
    });

    fireEvent.click(wrapper.getByText("Save"));

    await waitFor(() => {
      expect(updateScope.isDone()).toBe(true);
      expect(
        wrapper.getByText("Auth wall configuration updated successfully.")
      ).toBeTruthy();
    });
  });
});
