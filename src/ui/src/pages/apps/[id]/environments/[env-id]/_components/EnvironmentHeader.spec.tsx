import { RenderResult, waitFor } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router";
import { render } from "@testing-library/react";
import { AppContext } from "~/pages/apps/[id]/App.context";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import { mockFetchStatus } from "~/testing/nocks/nock_environment";
import EnvironmentHeader from "./EnvironmentHeader";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import nock from "nock/types";

interface WrapperProps {
  app?: App;
  environments?: Environment[];
}

describe("~/pages/apps/[id]/environments/[env-id]/_components/EnvironmentHeader.tsx", () => {
  let wrapper: RenderResult;
  const defaultApp = mockApp();
  const defaultEnvs = mockEnvironments({ app: defaultApp });

  const createWrapper = ({
    app = defaultApp,
    environments = defaultEnvs,
  }: WrapperProps) => {
    const memoryRouter = createMemoryRouter([
      {
        path: "*",
        element: (
          <AppContext.Provider
            value={{ app, environments, setRefreshToken: () => {} }}
          >
            <EnvironmentContext.Provider
              value={{ environment: environments[0] }}
            >
              <EnvironmentHeader />
            </EnvironmentContext.Provider>
          </AppContext.Provider>
        ),
      },
    ]);

    wrapper = render(<RouterProvider router={memoryRouter} />);
  };

  describe("when not yet published", () => {
    beforeEach(() => {
      createWrapper({});
    });

    test("should display the header properly", () => {
      expect(wrapper.container.innerHTML).toContain("PublicOffIcon");
      expect(wrapper.getByText("app.stormkit.io")).toBeTruthy();
      expect(wrapper.getByText("master")).toBeTruthy();
      expect(wrapper.getByText("Not yet published")).toBeTruthy();
    });
  });

  describe("when not yet deployed", () => {
    beforeEach(() => {
      const envs = [...defaultEnvs];
      envs[0].lastDeploy = undefined;
      createWrapper({ environments: envs });
    });

    test("should display the header properly", () => {
      expect(wrapper.container.innerHTML).toContain("PublicOffIcon");
      expect(wrapper.getByText("app.stormkit.io")).toBeTruthy();
      expect(wrapper.getByText("master")).toBeTruthy();
      expect(wrapper.getByText("Not yet deployed")).toBeTruthy();
    });
  });

  describe("when deployed and published with status 200", () => {
    let scope: nock.Scope;

    beforeEach(() => {
      const envs = [...defaultEnvs];
      envs[0].lastDeploy = { id: "1231231", createdAt: Date.now(), exit: 0 };
      envs[0].published = [
        { deploymentId: "682381", branch: "main", percentage: 100 },
      ];

      scope = mockFetchStatus({
        appId: defaultApp.id,
        url: envs[0].preview,
        response: { status: 200 },
      });

      createWrapper({ environments: envs });
    });

    test("should display the header properly", async () => {
      expect(wrapper.container.innerHTML).toContain("spinner");
      expect(wrapper.getByText("app.stormkit.io")).toBeTruthy();
      expect(wrapper.getByText("master")).toBeTruthy();

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(wrapper.getByText("200")).toBeTruthy();
        expect(wrapper.getByText("Published"));
        expect(wrapper.getByText("682381"));
        expect(wrapper.container.innerHTML).toContain("PublicIcon");
      });
    });
  });

  describe("when deployed and published with status 404", () => {
    let scope: nock.Scope;

    beforeEach(() => {
      const envs = [...defaultEnvs];
      envs[0].lastDeploy = { id: "1231231", createdAt: Date.now(), exit: 0 };
      envs[0].published = [
        { deploymentId: "682381", branch: "main", percentage: 100 },
      ];

      scope = mockFetchStatus({
        appId: defaultApp.id,
        url: envs[0].preview,
        response: { status: 404 },
      });

      createWrapper({ environments: envs });
    });

    test("should display the header properly", async () => {
      expect(wrapper.container.innerHTML).toContain("spinner");
      expect(wrapper.getByText("app.stormkit.io")).toBeTruthy();
      expect(wrapper.getByText("master")).toBeTruthy();

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(wrapper.getByText("404")).toBeTruthy();
        expect(wrapper.getByText("Published"));
        expect(wrapper.getByText("682381"));
        expect(wrapper.getByLabelText("Deployment not found"));
        expect(wrapper.container.innerHTML).toContain("PublicOffIcon");
      });
    });
  });
});
