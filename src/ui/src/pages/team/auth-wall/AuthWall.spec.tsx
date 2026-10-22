import type { RenderResult } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { render, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { AuthContext } from "~/pages/auth/Auth.context";
import { RootContext } from "~/pages/Root.context";
import mockTeams from "~/testing/data/mock_teams";
import {
  mockFetchTeamEnvs,
  mockFetchTeamLogins,
  mockUpdateEnvAuthWall,
} from "~/testing/nocks/nock_team_auth_wall";
import AuthWall from "./AuthWall";

describe("~/pages/team/auth-wall/AuthWall.tsx", () => {
  let wrapper: RenderResult;
  const teams = mockTeams();

  const createWrapper = ({
    team,
    edition = "enterprise",
  }: {
    team: Team;
    edition?: Edition;
  }) => {
    wrapper = render(
      <RootContext.Provider
        value={{
          mode: "dark",
          setMode: () => {},
          details: { license: { seats: 1, remaining: 1, edition } },
        }}
      >
        <AuthContext.Provider value={{ teams }}>
          <MemoryRouter initialEntries={[`/${team.slug}/auth-wall`]}>
            <Routes>
              <Route path="/:team/auth-wall" element={<AuthWall />} />
            </Routes>
          </MemoryRouter>
        </AuthContext.Provider>
      </RootContext.Provider>,
    );
  };

  it("shows developers the environments, without tabs", async () => {
    const team = teams.find(t => t.currentUserRole === "developer")!;
    const scope = mockFetchTeamEnvs({ teamId: team.id });

    createWrapper({ team });

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(
        wrapper.getByRole("heading", { name: "Environments" }),
      ).toBeTruthy();
    });

    expect(() => wrapper.getByRole("tab", { name: "Logins" })).toThrow();
  });

  it.each(["owner", "admin"])("shows the logins tab to %s", async role => {
    const team = teams.find(t => t.currentUserRole === role)!;
    const envs = mockFetchTeamEnvs({ teamId: team.id });

    createWrapper({ team });

    await waitFor(() => {
      expect(envs.isDone()).toBe(true);
      expect(wrapper.getByRole("tab", { name: "Environments" })).toBeTruthy();
    });

    const logins = mockFetchTeamLogins({ teamId: team.id });
    fireEvent.click(wrapper.getByRole("tab", { name: "Logins" }));

    await waitFor(() => {
      expect(logins.isDone()).toBe(true);
      expect(
        wrapper.getByText(
          "This team has no logins yet. Click the button above to add one.",
        ),
      ).toBeTruthy();
    });
  });

  // The environments tab is rebuilt when shown again, so it must not keep
  // showing the modes from before an update.
  it("shows updated modes after switching tabs", async () => {
    const team = teams.find(t => t.currentUserRole === "owner")!;
    const app = (authWall: "" | "all") => [
      {
        id: "1",
        displayName: "storefront",
        environments: [{ id: "11", name: "production", authWall }],
      },
    ];

    mockFetchTeamEnvs({ teamId: team.id, apps: app("") });
    createWrapper({ team });

    await waitFor(() => {
      expect(wrapper.getByText("storefront")).toBeTruthy();
    });

    const update = mockUpdateEnvAuthWall({
      teamId: team.id,
      envId: "11",
      authWall: "all",
    });

    const reload = mockFetchTeamEnvs({ teamId: team.id, apps: app("all") });

    fireEvent.click(wrapper.getByText("storefront"));
    fireEvent.click(wrapper.getByText("Apply"));

    await waitFor(() => {
      expect(update.isDone()).toBe(true);
      expect(reload.isDone()).toBe(true);
    });

    mockFetchTeamLogins({ teamId: team.id });
    fireEvent.click(wrapper.getByRole("tab", { name: "Logins" }));
    fireEvent.click(wrapper.getByRole("tab", { name: "Environments" }));

    await waitFor(() => {
      expect(wrapper.getByText("All URLs, including domains")).toBeTruthy();
    });
  });

  it("asks community users to upgrade", () => {
    createWrapper({ team: teams[0], edition: "community" });

    expect(
      wrapper.getByText(
        "You need to upgrade your plan to access this feature.",
      ),
    ).toBeTruthy();
  });
});
