import {
  RenderResult,
  fireEvent,
  waitFor,
  getByLabelText,
} from "@testing-library/react";
import { describe, expect, beforeEach, it } from "vitest";
import { AppContext } from "~/pages/apps/[id]/App.context";
import { RootContext } from "~/pages/Root.context";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import { renderWithRouter } from "~/testing/helpers";
import EnvMenu from "./EnvMenu";

declare const global: {
  NavigateMock: any;
};

type Edition = "development" | "self-hosted" | "cloud";

interface WrapperProps {
  app?: App;
  environments?: Environment[];
  edition?: Edition;
  setRefreshToken?: () => void;
}

describe("~/layouts/AppLayout/EnvMenu.tsx", () => {
  let wrapper: RenderResult;
  const defaultApp = mockApp();
  const defaultEnvs = mockEnvironments({ app: defaultApp });

  const createWrapper = ({
    app = defaultApp,
    environments = defaultEnvs,
    edition = "self-hosted",
    setRefreshToken = () => {},
  }: WrapperProps) => {
    wrapper = renderWithRouter({
      path: "/apps/:id/environments/:envId",
      initialEntries: [`/apps/${app.id}/environments/${environments[0].id}`],
      el: () => (
        <RootContext.Provider
          value={{
            mode: "dark",
            setMode: () => {},
            details: { stormkit: { apiCommit: "", apiVersion: "", edition } },
          }}
        >
          <AppContext.Provider value={{ app, environments, setRefreshToken }}>
            <EnvMenu />
          </AppContext.Provider>
        </RootContext.Provider>
      ),
    });
  };

  beforeEach(() => {
    createWrapper({});
  });

  it("should have the environment selector rendered correctly", async () => {
    const select = wrapper.getByLabelText("Environment selector");
    expect(select).toBeTruthy();
    fireEvent.mouseDown(select.firstChild!);

    await waitFor(() => {
      expect(
        getByLabelText(document.body, "development environment"),
      ).toBeTruthy();
    });

    fireEvent.click(getByLabelText(document.body, "development environment"));

    await waitFor(() => {
      expect(global.NavigateMock).toHaveBeenCalledWith(
        `/apps/1/environments/${defaultEnvs[1].id}`,
      );
    });
  });

  it("should render menu links", () => {
    const links = wrapper
      .getAllByRole("link")
      .map(link => link.getAttribute("href"));

    expect(links).toEqual([
      `/apps/${defaultApp.id}/environments/${defaultEnvs[0].id}`,
      `/apps/${defaultApp.id}/environments/${defaultEnvs[0].id}/deployments`,
      `/apps/${defaultApp.id}/environments/${defaultEnvs[0].id}/snippets`,
      `/apps/${defaultApp.id}/environments/${defaultEnvs[0].id}/function-triggers`,
      `/apps/${defaultApp.id}/environments/${defaultEnvs[0].id}/volumes`,
      `/apps/${defaultApp.id}/environments/${defaultEnvs[0].id}/database`,
      `/apps/${defaultApp.id}/environments/${defaultEnvs[0].id}/auth`,
      `/apps/${defaultApp.id}/environments/${defaultEnvs[0].id}/mailer`,
      `/apps/${defaultApp.id}/environments/${defaultEnvs[0].id}/analytics`,
    ]);
  });

  // Auth requires a database and is self-hosted only — both cloud and the
  // development edition must hide it.
  it.each<Edition>(["cloud", "development"])(
    "hides the Authentication link on the %s edition",
    edition => {
      wrapper.unmount(); // drop the self-hosted render from beforeEach
      createWrapper({ edition });

      const links = wrapper
        .getAllByRole("link")
        .map(link => link.getAttribute("href"));

      expect(links).not.toContain(
        `/apps/${defaultApp.id}/environments/${defaultEnvs[0].id}/auth`,
      );
      expect(wrapper.queryByText("Authentication")).toBeNull();
    },
  );
});
