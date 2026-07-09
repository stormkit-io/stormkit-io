import { beforeEach, describe, expect, it } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { AuthContext } from "~/pages/auth/Auth.context";
import mockApp from "~/testing/data/mock_app";
import mockTeams from "~/testing/data/mock_teams";
import mockUser from "~/testing/data/mock_user";
import { mockFetchTeamApps } from "~/testing/nocks/nock_app";
import LocalStorage from "~/utils/storage/LocalStorage";
import AppSelector from "./AppSelector";

describe("~/layouts/AppLayout/AppSelector.tsx", () => {
  let currentApp: App;
  let teams: Team[];
  let team: Team;

  const createWrapper = () => {
    currentApp = mockApp();
    teams = mockTeams();
    team = teams[0];

    render(
      <AuthContext.Provider value={{ user: mockUser(), teams }}>
        <AppSelector app={currentApp} team={team} />
      </AuthContext.Provider>,
    );
  };

  beforeEach(() => {
    localStorage.clear();
  });

  it("should record the visited app in recent apps", () => {
    createWrapper();

    expect(LocalStorage.get("recent_apps")).toEqual([
      {
        id: currentApp.id,
        displayName: currentApp.displayName,
        repo: currentApp.repo,
      },
    ]);
  });

  it("should list team apps when the arrow is clicked", async () => {
    const otherApp = mockApp({ id: "6421", displayName: "other-app" });

    createWrapper();

    const scope = mockFetchTeamApps({
      teamId: team.id,
      response: { apps: [currentApp, otherApp], hasNextPage: false },
    });

    fireEvent.click(screen.getByLabelText("Select app"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(screen.getByText("other-app")).toBeTruthy();
    });

    expect(
      screen.getByText("other-app").closest("a")?.getAttribute("href"),
    ).toBe("/apps/6421");
  });

  it("should list apps from another team when switched", async () => {
    createWrapper();

    mockFetchTeamApps({
      teamId: team.id,
      response: { apps: [currentApp], hasNextPage: false },
    });

    fireEvent.click(screen.getByLabelText("Select app"));

    await waitFor(() => {
      expect(screen.getByRole("combobox")).toBeTruthy();
    });

    const otherTeamApp = mockApp({ id: "8151", displayName: "other-team-app" });

    const scope = mockFetchTeamApps({
      teamId: teams[1].id,
      response: { apps: [otherTeamApp], hasNextPage: false },
    });

    fireEvent.mouseDown(screen.getByRole("combobox"));
    fireEvent.click(screen.getByText(teams[1].name));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(screen.getByText("other-team-app")).toBeTruthy();
    });
  });

  it("should list recently visited apps excluding the current app", async () => {
    LocalStorage.set("recent_apps", [
      {
        id: "9911",
        displayName: "recent-app",
        repo: "github/stormkit-io/recent-app",
      },
    ]);

    createWrapper();

    mockFetchTeamApps({
      teamId: team.id,
      response: { apps: [], hasNextPage: false },
    });

    fireEvent.click(screen.getByLabelText("Select app"));

    await waitFor(() => {
      expect(screen.getByText("Recent apps")).toBeTruthy();
      expect(screen.getByText("recent-app")).toBeTruthy();
    });

    expect(
      screen.getByText("recent-app").closest("a")?.getAttribute("href"),
    ).toBe("/apps/9911");

    expect(screen.queryByTestId(`recent-app-${currentApp.id}`)).toBeNull();
  });

  it("should exclude legacy entries recorded without repo info", async () => {
    LocalStorage.set("recent_apps", [{ id: "9911", displayName: "legacy-app" }]);

    createWrapper();

    mockFetchTeamApps({
      teamId: team.id,
      response: { apps: [], hasNextPage: false },
    });

    fireEvent.click(screen.getByLabelText("Select app"));

    await waitFor(() => {
      expect(screen.queryByText("legacy-app")).toBeNull();
      expect(screen.getByText("No recently visited apps.")).toBeTruthy();
    });
  });

  it("should display a message when there are no recent apps", async () => {
    createWrapper();

    mockFetchTeamApps({
      teamId: team.id,
      response: { apps: [], hasNextPage: false },
    });

    fireEvent.click(screen.getByLabelText("Select app"));

    await waitFor(() => {
      expect(screen.getByText("No recently visited apps.")).toBeTruthy();
    });
  });
});
