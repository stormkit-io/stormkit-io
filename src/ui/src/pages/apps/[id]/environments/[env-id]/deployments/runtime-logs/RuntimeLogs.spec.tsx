import type { Scope } from "nock";
import { fireEvent, type RenderResult, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AppContext } from "~/pages/apps/[id]/App.context";
import mockApp from "~/testing/data/mock_app";
import mockDeployment from "~/testing/data/mock_deployment";
import mockEnvironment from "~/testing//data/mock_environment";
import { mockFetchDeploymentLogs } from "~/testing/nocks/nock_logs";
import RuntimeLogs from "./RuntimeLogs";
import { renderWithRouter } from "~/testing/helpers";

interface Props {
  app?: App;
  deployment?: DeploymentV2;
  environment?: Environment;
  hasNextPage?: boolean;
}

describe("~/pages/apps/[id]/environments/[env-id]/deployments/runtime-logs/RuntimeLogs.spec.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;
  let currentDeploy: DeploymentV2;
  let scope: Scope;
  let id = 1050101;

  const createWrapper = ({
    app,
    environment,
    deployment,
    hasNextPage = false,
  }: Props | undefined = {}) => {
    currentApp = app || mockApp();
    currentEnv = environment || mockEnvironment({ app: currentApp });
    currentDeploy =
      deployment ||
      mockDeployment({
        appId: currentApp.id,
        envId: currentEnv.id,
        id: id++ + "",
      });

    scope = mockFetchDeploymentLogs({
      appId: currentApp.id,
      deploymentId: currentDeploy.id,
      response: {
        logs: [
          {
            id: "123",
            appId: "592128846360",
            envId: "483571891194",
            deploymentId: "70884402118696",
            data: "START RequestId: 7c64e362-004c-40f8-a325-ef9af252cd8d Version: 1\n2022-10-19T16:54:01.246Z\t7c64e362-004c-40f8-a325-ef9af252cd8d\tINFO\tsample log from sample-project 1\nEND RequestId: 7c64e362-004c-40f8-a325-ef9af252cd8d\nREPORT RequestId: 7c64e362-004c-40f8-a325-ef9af252cd8d\tDuration: 283.86 ms\tBilled Duration: 284 ms\tMemory Size: 512 MB\tMax Memory Used: 66 MB\tInit Duration: 150.46 ms\t\n",
            timestamp: "1666198441",
          },
          {
            id: "234",
            appId: "592128846360",
            envId: "483571891194",
            deploymentId: "70884402118696",
            data: "START RequestId: bd9c372e-3469-46e8-ae51-abe40174ef05 Version: 1\n2022-10-19T13:53:24.661Z\tbd9c372e-3469-46e8-ae51-abe40174ef05\tINFO\tsample log from sample-project 2\nEND RequestId: bd9c372e-3469-46e8-ae51-abe40174ef05\nREPORT RequestId: bd9c372e-3469-46e8-ae51-abe40174ef05\tDuration: 14.53 ms\tBilled Duration: 15 ms\tMemory Size: 512 MB\tMax Memory Used: 69 MB\t\n",
            timestamp: "1666187604",
          },
          {
            id: "456",
            appId: "592128846360",
            envId: "483571891194",
            deploymentId: "70884402118696",
            data: "START RequestId: 3c316656-f979-4eaf-a373-6fb7f4de4634 Version: 1\n2022-10-19T13:49:30.129Z\t3c316656-f979-4eaf-a373-6fb7f4de4634\tINFO\tsample log from sample-project 3\nEND RequestId: 3c316656-f979-4eaf-a373-6fb7f4de4634\nREPORT RequestId: 3c316656-f979-4eaf-a373-6fb7f4de4634\tDuration: 65.49 ms\tBilled Duration: 66 ms\tMemory Size: 512 MB\tMax Memory Used: 69 MB\t\n",
            timestamp: "1666187370",
          },
        ],
        hasNextPage,
      },
    });

    wrapper = renderWithRouter({
      path: "/apps/:appId/environments/:envId/deployments/:deploymentId/runtime-logs",
      initialEntries: [
        `/apps/${currentApp.id}/environments/${currentEnv.id}/deployments/${currentDeploy.id}/runtime-logs`,
      ],
      el: () => (
        <AppContext.Provider
          value={{
            app: currentApp,
            environments: [currentEnv],
            setRefreshToken: vi.fn(),
          }}
        >
          <RuntimeLogs />
        </AppContext.Provider>
      ),
    });
  };

  it("should contain a link back to the deployment page", async () => {
    createWrapper();

    await waitFor(() => {
      expect(wrapper.getByText("Runtime logs")).toBeTruthy();
      expect(
        wrapper.getByText(`#${currentDeploy.id}`).getAttribute("href")
      ).toBe("/apps/1/environments/1429333243019/deployments/1050101");
    });
  });

  it("should display logs", async () => {
    createWrapper();

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(
        wrapper.getByText("sample log from sample-project 1")
      ).toBeTruthy();
      expect(
        wrapper.getByText("sample log from sample-project 2")
      ).toBeTruthy();
      expect(
        wrapper.getByText("sample log from sample-project 3")
      ).toBeTruthy();
    });
  });

  describe("pagination", () => {
    it("should paginate when load more is clicked", async () => {
      createWrapper({ hasNextPage: true });

      await waitFor(() => {
        expect(wrapper.getByText("Load more")).toBeTruthy();
      });

      const paginationScope = mockFetchDeploymentLogs({
        appId: currentApp.id,
        deploymentId: currentDeploy.id,
        keySetId: "456",
        response: {
          hasNextPage: false,
          logs: [
            {
              id: "123",
              appId: currentApp.id,
              envId: currentEnv.id!,
              deploymentId: currentDeploy.id,
              data: "Hello from a second page log",
              timestamp: "1666198441",
            },
          ],
        },
      });

      fireEvent.click(wrapper.getByText("Load more"));

      await waitFor(() => {
        expect(paginationScope.isDone()).toBe(true);
        expect(wrapper.getByText("Hello from a second page log")).toBeTruthy();
      });
    });

    it("should change sort order when sort button is clicked", async () => {
      createWrapper({ hasNextPage: true });

      await waitFor(() => {
        expect(wrapper.getByText("Load more")).toBeTruthy();
        expect(wrapper.getByText("Descending")).toBeTruthy();
      });

      const paginationScope = mockFetchDeploymentLogs({
        appId: currentApp.id,
        deploymentId: currentDeploy.id,
        sort: "asc",
        response: {
          hasNextPage: false,
          logs: [
            {
              id: "123",
              appId: currentApp.id,
              envId: currentEnv.id!,
              deploymentId: currentDeploy.id,
              data: "Hello from sorted logs",
              timestamp: "1666198441",
            },
          ],
        },
      });

      fireEvent.click(wrapper.getByText("Descending"));

      await waitFor(() => {
        expect(paginationScope.isDone()).toBe(true);
        expect(wrapper.getByText("Hello from sorted logs")).toBeTruthy();
      });
    });
  });
});
