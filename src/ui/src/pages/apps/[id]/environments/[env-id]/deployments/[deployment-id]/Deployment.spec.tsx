import type { Scope } from "nock";
import { describe, beforeEach, afterEach, expect, it, vi } from "vitest";
import { MemoryRouter, Routes, Route } from "react-router";
import { RenderResult, waitFor, render, act } from "@testing-library/react";
import { AppContext } from "~/pages/apps/[id]/App.context";
import mockApp from "~/testing/data/mock_app";
import mockDeployments from "~/testing/data/mock_deployments_v2";
import { mockFetchDeployments } from "~/testing/nocks/nock_deployments_v2";
import Deployment from "./Deployment";

interface Props {
  deployment?: DeploymentV2;
}

vi.mock("~/utils/helpers/deployments", () => ({
  formattedDate: () => "21.09.2022 - 21:30",
}));

describe("~/apps/[id]/environments/[env-id]/deployments/Deployment.tsx", () => {
  let wrapper: RenderResult;
  let currentDeploy: DeploymentV2;
  let currentApp: App;
  let scope: Scope;
  let setRefreshToken = vi.fn();

  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
  });

  afterEach(() => {
    wrapper?.unmount();
    vi.useRealTimers();
  });

  const createWrapper = ({ deployment }: Props | undefined = {}) => {
    currentApp = mockApp();
    currentDeploy = deployment || mockDeployments()[0];

    scope = mockFetchDeployments({
      deploymentId: currentDeploy.id,
      response: { deployments: [currentDeploy] },
    });

    wrapper = render(
      <MemoryRouter initialEntries={[`/${currentDeploy.id}`]} initialIndex={0}>
        <Routes>
          <Route
            path="/:deploymentId"
            element={
              <AppContext.Provider
                value={{
                  app: currentApp,
                  environments: [],
                  setRefreshToken,
                }}
              >
                <Deployment />
              </AppContext.Provider>
            }
          />
        </Routes>
      </MemoryRouter>,
    );
  };

  it("should continuously poll the deployment until it is not running anymore", async () => {
    const deployment = mockDeployments()[0];
    deployment.status = "running";

    createWrapper({ deployment });

    // Wait for the first fetch to complete
    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });

    // Set up a second mock for the polling request (nock consumes interceptors)
    const secondScope = mockFetchDeployments({
      deploymentId: deployment.id,
      response: { deployments: [{ ...deployment, status: "success" }] },
    });

    // Advance timers to trigger the polling interval (wrapped in act)
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });

    // Verify the second fetch was made
    await waitFor(() => {
      expect(secondScope.isDone()).toBe(true);
    });
  });

  it("should display deployment details", async () => {
    createWrapper();

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText("chore: update packages")).toBeTruthy();
      expect(wrapper.getByText("published")).toBeTruthy();
      expect(wrapper.getByText("21.09.2022 - 21:30")).toBeTruthy();
      expect(wrapper.getByText(/main/)).toBeTruthy();
    });
  });

  it("should display the logs", async () => {
    const deployment = mockDeployments()[0];
    deployment.logs = [
      {
        title: "npm run build",
        message: "Nuxt CLI v3.0.0-rc.8",
        status: true,
        payload: "",
      },
    ];

    createWrapper({ deployment });

    await waitFor(() => {
      expect(wrapper.getByText("npm run build")).toBeTruthy();
    });
  });

  it("should display the preview button for successful deployments", async () => {
    createWrapper();

    await waitFor(() => {
      expect(wrapper.getByText("Preview")).toBeTruthy();
    });
  });

  it("should display expand menu", async () => {
    createWrapper();

    await waitFor(() => {
      expect(
        wrapper.getByLabelText(`Deployment ${currentDeploy.id} menu`),
      ).toBeTruthy();
    });
  });
});
