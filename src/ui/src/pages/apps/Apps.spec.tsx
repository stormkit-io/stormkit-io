import type { RenderResult } from "@testing-library/react";
import type { Scope } from "nock";
import * as router from "react-router";
import { describe, it, beforeEach, expect, afterEach } from "vitest";
import { render, fireEvent, waitFor } from "@testing-library/react";
import { AuthContext } from "~/pages/auth/Auth.context";
import { mockFetchApps } from "~/testing/nocks/nock_app";
import mockApp from "~/testing/data/mock_app";
import mockTeams from "~/testing/data/mock_teams";
import mockUser from "~/testing/data/mock_user";
import { mockFetchTeamStats } from "~/testing/nocks/nock_team_stats";
import { LocalStorage } from "~/utils/storage";
import { LS_PROVIDER } from "~/utils/api/Api";
import Apps from "./Apps";

interface Props {
  provider?: Provider | null;
}

describe("~/pages/apps/Apps.tsx", () => {
  const teams = mockTeams();
  const apps = [
    mockApp({ id: "1231231", displayName: "My-app" }),
    mockApp({ id: "5121231", displayName: "My-second-app" }),
  ];

  let wrapper: RenderResult;

  const createWrapper = ({ provider = "github" }: Props = {}) => {
    if (provider) {
      LocalStorage.set(LS_PROVIDER, provider);
    }

    const memoryRouter = router.createMemoryRouter([
      {
        path: "*",
        element: (
          <AuthContext.Provider
            value={{
              user: mockUser(),
              teams,
              providers: {
                github: false,
                gitlab: false,
                bitbucket: false,
                [provider as Provider]: true,
              },
            }}
          >
            <Apps />
          </AuthContext.Provider>
        ),
      },
    ]);

    wrapper = render(<router.RouterProvider router={memoryRouter} />);
  };

  afterEach(() => {
    LocalStorage.del(LS_PROVIDER);
  });

  describe("when user has already created apps", () => {
    let scope: Scope;
    let teamStatsScope: Scope;

    beforeEach(() => {
      teamStatsScope = mockFetchTeamStats({ teamId: teams[0].id });

      scope = mockFetchApps({
        teamId: teams[0].id,
        response: { apps, hasNextPage: false },
      });

      createWrapper();
    });

    it("should fetch the list of apps", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("My-app")).toBeTruthy();
        expect(wrapper.getByText("My-second-app")).toBeTruthy();
      });
    });

    it("should have a button to create a new app", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("Create new app")).toBeTruthy();
      });
    });

    it("should have a button to create a new app from url", async () => {
      let button: HTMLElement;

      await waitFor(() => {
        button = wrapper.getByText("Create new app");
      });

      fireEvent.click(button!);

      await waitFor(() => {
        expect(
          wrapper
            .getByRole("link", { name: "Import from URL" })
            .getAttribute("href")
        ).toBe("/apps/new/url");
      });
    });

    it("should not have a button to load more", async () => {
      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(() => wrapper.getByText("Load more")).toThrow();
      });
    });

    it("makes a call to fetch team stats", async () => {
      await waitFor(() => {
        expect(teamStatsScope.isDone()).toBeTruthy();
      });
    });
  });

  describe("when user has more apps to load", () => {
    beforeEach(() => {
      mockFetchTeamStats({ teamId: teams[0].id });

      mockFetchApps({
        teamId: teams[0].id,
        response: { apps, hasNextPage: true },
      });

      createWrapper();
    });

    it("should have a button to load more", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("Load more")).toBeTruthy();
      });
    });

    it("should trigger a new call when the button is clicked", async () => {
      const newApps = [{ ...apps[0], id: "7481841" }];

      const scope = mockFetchApps({
        teamId: teams[0].id,
        from: 20,
        response: { apps: newApps, hasNextPage: false },
      });

      let button: HTMLElement;

      await waitFor(() => {
        button = wrapper.getByText("Load more");
      });

      fireEvent.click(button!);

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
      });
    });
  });

  describe("empty list - with provider", () => {
    let teamStatsScope: Scope;

    beforeEach(() => {
      teamStatsScope = mockFetchTeamStats({ teamId: teams[0].id });

      mockFetchApps({
        teamId: teams[0].id,
        response: { apps: [], hasNextPage: false },
      });
      createWrapper();
    });

    it("should display an empty list text", async () => {
      await waitFor(() => {
        expect(
          wrapper.getByText("Grant access to import your private repositories")
        ).toBeTruthy();

        // Should not fetch team stats when there are no apps
        expect(teamStatsScope.isDone()).toBe(false);
      });
    });
  });

  describe("empty list - without provider", () => {
    beforeEach(() => {
      mockFetchApps({
        teamId: teams[0].id,
        response: { apps: [], hasNextPage: false },
      });
      createWrapper({ provider: null });
    });

    it("should display an empty list text", async () => {
      await waitFor(() => {
        expect(
          wrapper.getByText(
            "Configure provider to import from private repositories"
          )
        ).toBeTruthy();
      });
    });
  });

  describe("filtering", () => {
    const findInput = () => wrapper.getAllByLabelText("Search apps").at(1);

    beforeEach(() => {
      mockFetchTeamStats({ teamId: teams[0].id });

      mockFetchApps({
        teamId: teams[0].id,
        response: { apps, hasNextPage: true },
      });

      createWrapper();
    });

    it("should submit a new request when search input is updated", async () => {
      await waitFor(() => {
        expect(findInput()).toBeTruthy();
        expect(wrapper.getByText("My-second-app")).toBeTruthy();
      });

      const scope = mockFetchApps({
        teamId: teams[0].id,
        filter: "hello",
        response: { apps: [apps[0]], hasNextPage: true },
      });

      fireEvent.change(findInput()!.querySelector("input")!, {
        target: { value: "hello" },
      });

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(() => wrapper.getByText("My-second-app")).toThrow();
      });
    });
  });
});
