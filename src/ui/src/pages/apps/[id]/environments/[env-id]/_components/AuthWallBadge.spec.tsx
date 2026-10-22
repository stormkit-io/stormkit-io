import type { RenderResult } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { render, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { AuthContext } from "~/pages/auth/Auth.context";
import { RootContext } from "~/pages/Root.context";
import mockApp from "~/testing/data/mock_app";
import mockEnvironment from "~/testing/data/mock_environment";
import mockTeams from "~/testing/data/mock_teams";
import { mockFetchAuthWallConfig } from "~/testing/nocks/nock_auth_wall";
import AuthWallBadge from "./AuthWallBadge";

describe("~/pages/apps/[id]/environments/[env-id]/_components/AuthWallBadge.tsx", () => {
  let wrapper: RenderResult;
  const teams = mockTeams();

  const app = mockApp();
  const env = mockEnvironment({ app });
  app.teamId = teams[2].id;

  const createWrapper = ({
    edition = "enterprise",
  }: { edition?: Edition } = {}) => {
    wrapper = render(
      <RootContext.Provider
        value={{
          mode: "dark",
          setMode: () => {},
          details: { license: { seats: 1, remaining: 1, edition } },
        }}
      >
        <AuthContext.Provider value={{ teams }}>
          <MemoryRouter>
            <AuthWallBadge app={app} environment={env} />
          </MemoryRouter>
        </AuthContext.Provider>
      </RootContext.Provider>,
    );
  };

  it.each`
    authwall | label
    ${""}    | ${"Auth Wall: Off"}
    ${"dev"} | ${"Auth Wall: Previews"}
    ${"all"} | ${"Auth Wall: All URLs"}
  `("shows $label and links to the team page", async ({ authwall, label }) => {
    const scope = mockFetchAuthWallConfig({
      appId: app.id,
      envId: env.id!,
      response: { authwall },
    });

    createWrapper();

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText(label)).toBeTruthy();
    });

    expect(wrapper.getByText(label).closest("a")?.getAttribute("href")).toBe(
      `/${teams[2].slug}/auth-wall?appId=${app.id}`,
    );
  });

  it("is hidden on the community edition", () => {
    createWrapper({ edition: "community" });
    expect(wrapper.container.textContent).toBe("");
  });

  // A slow response for the previous environment must not replace the mode
  // of the current one.
  it("shows the mode of the current environment after switching", async () => {
    const other = mockEnvironment({ app });
    other.id = "other-env";

    const slow = mockFetchAuthWallConfig({
      appId: app.id,
      envId: env.id!,
      response: { authwall: "all" },
      delay: 150,
    });

    const fast = mockFetchAuthWallConfig({
      appId: app.id,
      envId: other.id,
      response: { authwall: "" },
    });

    const providers = (environment: Environment) => (
      <RootContext.Provider
        value={{
          mode: "dark",
          setMode: () => {},
          details: {
            license: { seats: 1, remaining: 1, edition: "enterprise" },
          },
        }}
      >
        <AuthContext.Provider value={{ teams }}>
          <MemoryRouter>
            <AuthWallBadge app={app} environment={environment} />
          </MemoryRouter>
        </AuthContext.Provider>
      </RootContext.Provider>
    );

    wrapper = render(providers(env));
    wrapper.rerender(providers(other));

    await waitFor(() => {
      expect(fast.isDone()).toBe(true);
      expect(wrapper.getByText("Auth Wall: Off")).toBeTruthy();
    });

    // Let the slow response, delayed by 150ms, arrive.
    await new Promise(resolve => setTimeout(resolve, 300));
    expect(slow.isDone()).toBe(true);

    expect(wrapper.getByText("Auth Wall: Off")).toBeTruthy();
  });
});
