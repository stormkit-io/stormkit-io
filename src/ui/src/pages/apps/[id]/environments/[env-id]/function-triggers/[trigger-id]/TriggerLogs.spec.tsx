import type { RenderResult } from "@testing-library/react";
import type { Scope } from "nock";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { cleanup, fireEvent, render, waitFor } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router";
import { AppContext } from "~/pages/apps/[id]/App.context";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import { mockFetchTriggerLogs } from "~/testing/nocks/nock_function_triggers";
import mockApp from "~/testing/data/mock_app";
import mockEnvironment from "~/testing//data/mock_environment";
import TriggerLogs from "./TriggerLogs";

interface Props {
  app?: App;
  environment?: Environment;
  logs?: TriggerLog[];
}

describe("~/apps/[id]/environments/[env-id]/function-triggers/[trigger-id]/TriggerLogs.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;
  let scope: Scope;

  const createWrapper = ({ app, environment, logs = [] }: Props = {}) => {
    currentApp = app || mockApp();
    currentEnv = environment || mockEnvironment({ app: currentApp });

    const memoryRouter = createMemoryRouter(
      [
        {
          path: "/:triggerId",
          element: <TriggerLogs />,
        },
      ],
      {
        initialIndex: 0,
        initialEntries: [`/123`],
      }
    );

    scope = mockFetchTriggerLogs({
      appId: currentApp.id,
      envId: currentEnv?.id!,
      triggerId: "123",
      response: { logs },
    });

    wrapper = render(
      <AppContext.Provider
        value={{
          app: currentApp,
          environments: [currentEnv],
          setRefreshToken: vi.fn(),
        }}
      >
        <EnvironmentContext.Provider value={{ environment: currentEnv }}>
          <RouterProvider router={memoryRouter} />
        </EnvironmentContext.Provider>
      </AppContext.Provider>
    );
  };

  beforeEach(() => {
    createWrapper({
      logs: [
        {
          createdAt: 1734602569,
          request: {
            payload: `{ "hello": "world" }`,
            url: "https://api.example.com/trigger-1",
          },
          response: { code: 200 },
        },
        {
          createdAt: 1734602328,
          request: {
            payload: `{ "hi": "world" }`,
            url: "https://api.example.com/trigger-2",
          },
          response: { code: 302 },
        },
      ],
    });
  });

  it("renders without error", () => {
    expect(wrapper.getByText("Trigger logs")).toBeTruthy();
    expect(wrapper.getByText("Last 25 logs for this trigger")).toBeTruthy();
  });

  it("should display a loader when fetching logs", async () => {
    expect(wrapper.container.innerHTML).toContain("MuiLinearProgress-root");

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });

    expect(wrapper.container.innerHTML).not.toContain("MuiLinearProgress-root");
  });

  it("should display a loader when fetching logs", async () => {
    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(
        wrapper.getByText("https://api.example.com/trigger-1")
      ).toBeTruthy();
      expect(
        wrapper.getByText("https://api.example.com/trigger-2")
      ).toBeTruthy();
    });
  });

  it("should refetch the trigger logs when refresh button is clicked", async () => {
    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });

    const newScope = mockFetchTriggerLogs({
      appId: currentApp.id,
      envId: currentEnv?.id!,
      triggerId: "123",
      response: { logs: [] },
    });

    await waitFor(() => {
      fireEvent.click(wrapper.getByTestId("refresh-logs"));
      expect(newScope.isDone()).toBe(true);
    });
  });

  it("should render failed logs that have no response code", async () => {
    cleanup();
    createWrapper({
      logs: [
        {
          createdAt: 1734602569,
          request: { url: "https://api.example.com/trigger-fail" },
          response: { error: "dial tcp: connection refused" },
        },
      ],
    });

    await waitFor(() => {
      expect(
        wrapper.getByText("https://api.example.com/trigger-fail")
      ).toBeTruthy();
      expect(wrapper.getByText("ERR")).toBeTruthy();
    });

    fireEvent.click(wrapper.getAllByTestId("trigger-log").at(0)!);

    await waitFor(() => {
      expect(
        wrapper.getByText("dial tcp: connection refused")
      ).toBeTruthy();
    });
  });

  it("should open a drawer when a row is clicked", async () => {
    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getAllByTestId("trigger-log")).toBeTruthy();
    });

    fireEvent.click(wrapper.getAllByTestId("trigger-log").at(0)!);

    await waitFor(() => {
      expect(wrapper.getByText("Log details")).toBeTruthy();
      expect(wrapper.getByText("Request payload")).toBeTruthy();
      expect(wrapper.getByText(`{ "hello": "world" }`)).toBeTruthy();
    });

    // This is trigger log #2 so shouldn't exist
    expect(() => wrapper.getByText(`{ "hi": "world" }`)).toThrow();
  });
});
