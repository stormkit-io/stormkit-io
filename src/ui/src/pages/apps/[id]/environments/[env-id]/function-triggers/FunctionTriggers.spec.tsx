import type { Scope } from "nock";
import {
  fireEvent,
  render,
  RenderResult,
  waitFor,
} from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { RouterProvider, createMemoryRouter } from "react-router";
import { AppContext } from "~/pages/apps/[id]/App.context";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import mockApp from "~/testing/data/mock_app";
import mockFunctionTriggers, {
  mockTriggerLog,
} from "~/testing/data/mock_function_triggers";
import mockEnvironment from "~/testing//data/mock_environment";
import * as mockActions from "~/testing/nocks/nock_function_triggers";
import FunctionTriggers from "./FunctionTriggers";

const {
  mockFetchFunctionTriggers,
  mockDeleteFunctionTrigger,
  mockInvokeFunctionTrigger,
  mockFetchTriggerLogs,
} = mockActions;

interface Props {
  app?: App;
  environment?: Environment;
  triggers?: FunctionTrigger[];
}

describe("~/apps/[id]/environments/[env-id]/function-triggers/FunctionTriggers.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;
  let currentTriggers: FunctionTrigger[];
  let scope: Scope;

  const createWrapper = ({
    app,
    environment,
    triggers,
  }: Props | undefined = {}) => {
    currentApp = app || mockApp();
    currentEnv = environment || mockEnvironment({ app: currentApp });
    currentTriggers = triggers || mockFunctionTriggers();

    scope = mockFetchFunctionTriggers({
      appId: currentApp.id,
      envId: currentEnv.id!,
      response: { triggers: currentTriggers },
    });

    const memoryRouter = createMemoryRouter([
      {
        path: "*",
        element: (
          <AppContext.Provider
            value={{
              app: currentApp,
              environments: [currentEnv],
              setRefreshToken: vi.fn(),
            }}
          >
            <EnvironmentContext.Provider value={{ environment: currentEnv }}>
              <FunctionTriggers />
            </EnvironmentContext.Provider>
          </AppContext.Provider>
        ),
      },
    ]);

    wrapper = render(<RouterProvider router={memoryRouter} />);
  };

  it("should list function triggers", async () => {
    createWrapper();

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(
        wrapper.getByText("https://app.stormkit.io/api/test")
      ).toBeTruthy();
      expect(wrapper.getByText("2 0 2 * *")).toBeTruthy();
    });
  });

  it("should handle deleting function trigger", async () => {
    createWrapper();

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(
        wrapper.getByText("https://app.stormkit.io/api/test")
      ).toBeTruthy();
    });

    const deleteScope = mockDeleteFunctionTrigger({
      appId: currentApp.id,
      envId: currentEnv.id!,
      tfid: currentTriggers[0].id!,
    });

    fireEvent.click(wrapper.getAllByLabelText("expand").at(0)!);

    await waitFor(() => {
      expect(wrapper.getByText("Delete")).toBeTruthy();
    });

    fireEvent.click(wrapper.getByText("Delete"));

    await waitFor(() => {
      expect(wrapper.getByText("Yes, continue")).toBeTruthy();
    });

    const refetchScope = mockFetchFunctionTriggers({
      appId: currentApp.id,
      envId: currentEnv.id!,
      response: { triggers: currentTriggers },
    });

    fireEvent.click(wrapper.getByText("Yes, continue"));

    await waitFor(() => {
      expect(deleteScope.isDone()).toBe(true);
      expect(refetchScope.isDone()).toBe(true);
    });
  });

  it("should handle running a trigger immediately", async () => {
    createWrapper();

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(
        wrapper.getByText("https://app.stormkit.io/api/test")
      ).toBeTruthy();
    });

    const invokeScope = mockInvokeFunctionTrigger({
      appId: currentApp.id,
      envId: currentEnv.id!,
      tfid: currentTriggers[0].id!,
    });

    fireEvent.click(wrapper.getAllByLabelText("expand").at(0)!);

    await waitFor(() => {
      expect(wrapper.getByText("Trigger now")).toBeTruthy();
    });

    fireEvent.click(wrapper.getByText("Trigger now"));

    await waitFor(() => {
      expect(invokeScope.isDone()).toBe(true);
      expect(wrapper.getByText("Log details")).toBeTruthy();
      // The response body is pretty printed.
      expect(
        wrapper.getByText(`{\n  "status": "ok"\n}`, { normalizer: s => s })
      ).toBeTruthy();
    });

    // The past runs are only fetched once the user navigates back to them.
    const logsScope = mockFetchTriggerLogs({
      appId: currentApp.id,
      envId: currentEnv.id!,
      triggerId: currentTriggers[0].id!,
      response: { logs: [mockTriggerLog()] },
    });

    expect(logsScope.isDone()).toBe(false);

    fireEvent.click(wrapper.getByTestId("back-to-logs"));

    await waitFor(() => {
      expect(logsScope.isDone()).toBe(true);
      expect(wrapper.getByText("Past triggers")).toBeTruthy();
    });
  });

  it("should open the past triggers drawer when the url is clicked", async () => {
    createWrapper();

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });

    const logsScope = mockFetchTriggerLogs({
      appId: currentApp.id,
      envId: currentEnv.id!,
      triggerId: currentTriggers[0].id!,
      response: { logs: [mockTriggerLog()] },
    });

    fireEvent.click(wrapper.getByTestId("trigger-url"));

    await waitFor(() => {
      expect(logsScope.isDone()).toBe(true);
      expect(wrapper.getByText("Past triggers")).toBeTruthy();
      expect(wrapper.getAllByTestId("trigger-log")).toHaveLength(1);
    });

    fireEvent.click(wrapper.getAllByTestId("trigger-log").at(0)!);

    await waitFor(() => {
      expect(wrapper.getByText("Log details")).toBeTruthy();
      expect(
        wrapper.getByText(`{\n  "status": "ok"\n}`, { normalizer: s => s })
      ).toBeTruthy();
    });

    fireEvent.click(wrapper.getByTestId("back-to-logs"));

    await waitFor(() => {
      expect(wrapper.getByText("Past triggers")).toBeTruthy();
    });
  });

  it("should render logs that failed without a response code", async () => {
    createWrapper();

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });

    const logsScope = mockFetchTriggerLogs({
      appId: currentApp.id,
      envId: currentEnv.id!,
      triggerId: currentTriggers[0].id!,
      response: {
        logs: [
          {
            createdAt: 1734602569,
            request: { url: "https://api.example.com/trigger-fail" },
            response: { error: "dial tcp: connection refused" },
          },
        ],
      },
    });

    fireEvent.click(wrapper.getByTestId("trigger-url"));

    await waitFor(() => {
      expect(logsScope.isDone()).toBe(true);
      expect(wrapper.getAllByText("ERR").length).toBeGreaterThan(0);
    });

    fireEvent.click(wrapper.getAllByTestId("trigger-log").at(0)!);

    await waitFor(() => {
      expect(wrapper.getByText("dial tcp: connection refused")).toBeTruthy();
    });
  });

  it("displays an empty list when there are no function triggers", async () => {
    createWrapper({ triggers: [] });

    await waitFor(() => {
      expect(wrapper.getByText(/It\'s quite empty in here\./)).toBeTruthy();
    });
  });
});
