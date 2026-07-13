import type { Scope } from "nock";
import { describe, expect, beforeEach, it, vi } from "vitest";
import { fireEvent, waitFor, type RenderResult } from "@testing-library/react";
import { AuthContext } from "~/pages/auth/Auth.context";
import { AppContext } from "~/pages/apps/[id]/App.context";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import mockTeams from "~/testing/data/mock_teams";
import mockUser from "~/testing/data/mock_user";
import mockApp from "~/testing/data/mock_app";
import mockEnv from "~/testing/data/mock_environment";
import {
  mockFetchVolumesConfig,
  mockFetchFiles,
} from "~/testing/nocks/nock_volumes";
import Volumes from "./Volumes";
import { renderWithRouter } from "~/testing/helpers";

interface Props {
  user: User;
}

describe("~/pages/apps/[id]/environments/[env-id]/volumes/Volumes.tsx", () => {
  let wrapper: RenderResult;
  let currentUser: User;
  let currentApp: App;
  let currentEnv: Environment;
  let fetchFilesScope: Scope;

  const createWrapper = ({ user }: Props) => {
    const teams = mockTeams();
    const setRefreshToken = vi.fn();
    currentApp = mockApp();
    currentEnv = mockEnv({ app: currentApp });

    fetchFilesScope = mockFetchFiles({
      appId: currentApp.id,
      envId: currentEnv.id!,
      beforeId: "",
      response: { files: [] },
    });

    wrapper = renderWithRouter({
      el: () => (
        <AuthContext.Provider value={{ user, teams }}>
          <AppContext.Provider
            value={{
              app: currentApp,
              setRefreshToken,
              environments: [currentEnv],
            }}
          >
            <EnvironmentContext.Provider value={{ environment: currentEnv }}>
              <Volumes />
            </EnvironmentContext.Provider>
          </AppContext.Provider>
        </AuthContext.Provider>
      ),
    });
  };

  describe("when user is an instance admin", () => {
    beforeEach(() => {
      currentUser = mockUser();
      currentUser.isAdmin = true;
    });

    describe("and volumes is not configured", () => {
      beforeEach(async () => {
        const scope = mockFetchVolumesConfig({
          response: { config: false },
          status: 200,
        });

        createWrapper({ user: currentUser });

        await waitFor(() => {
          expect(scope.isDone()).toBe(true);
        });
      });

      it("should display an empty page with a configure button", async () => {
        await waitFor(() => {
          expect(
            wrapper.getByText(
              "Persist your files seamlessly using Stormkit Volumes"
            )
          ).toBeTruthy();
        });

        fireEvent.click(wrapper.getByText("Configure"));

        await waitFor(() => {
          expect(wrapper.findByText("Configure Stormkit Volumes")).toBeTruthy();
        });
      });

      it("should display a learn more button", async () => {
        await waitFor(() => {
          expect(wrapper.getByText("Learn more").getAttribute("href")).toBe(
            "https://www.stormkit.io/docs/features/volumes"
          );
        });
      });
    });

    describe("and volumes is configured", () => {
      beforeEach(async () => {
        const scope = mockFetchVolumesConfig({
          response: { config: true },
          status: 200,
        });

        createWrapper({ user: currentUser });

        await waitFor(() => {
          expect(scope.isDone()).toBe(true);
        });
      });

      it("should not display an empty page with a configure button", async () => {
        await waitFor(() => {
          // This should no longer be rendered
          expect(() =>
            wrapper.getByText(
              "Persist your files seamlessly using Stormkit Volumes"
            )
          ).toThrow();

          expect(() => {
            expect(wrapper.getByText("No files uploaded yet")).toBeTruthy();
          });
        });
      });
    });
  });

  describe("when user is not an instance admin", () => {
    beforeEach(() => {
      currentUser = mockUser();
      currentUser.isAdmin = false;
    });

    describe("and volumes is not configured", () => {
      beforeEach(async () => {
        const scope = mockFetchVolumesConfig({
          response: { config: false },
          status: 200,
        });

        createWrapper({ user: currentUser });

        await waitFor(() => {
          expect(scope.isDone()).toBe(true);
        });
      });

      it("should display an empty page", async () => {
        await waitFor(() => {
          expect(
            wrapper.getByText(
              /Volumes is not configured for this Stormkit instance\./
            )
          ).toBeTruthy();

          expect(
            wrapper.getByText(
              /Contact your administrator for more information\./
            )
          ).toBeTruthy();
        });

        expect(() => wrapper.getByText("Configure")).toThrow();
      });

      it("should display a learn more button", async () => {
        await waitFor(() => {
          expect(wrapper.getByText("Learn more").getAttribute("href")).toBe(
            "https://www.stormkit.io/docs/features/volumes"
          );
        });
      });
    });

    describe("and volumes is configured", () => {
      beforeEach(async () => {
        const scope = mockFetchVolumesConfig({
          response: { config: true },
          status: 200,
        });

        createWrapper({ user: currentUser });

        await waitFor(() => {
          expect(scope.isDone()).toBe(true);
        });
      });

      it("should display empty list", async () => {
        // This should not be rendered anymore
        expect(() =>
          wrapper.getByText(
            "Persist your files seamlessly using Stormkit Volumes"
          )
        ).toThrow();

        await waitFor(() => {
          expect(wrapper.getByText("No files uploaded yet")).toBeTruthy();
          expect(fetchFilesScope.isDone()).toBe(true);
        });
      });
    });
  });
});
