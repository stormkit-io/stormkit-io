import type { RenderResult } from "@testing-library/react";
import type { AuthContextProps } from "~/pages/auth/Auth.context";
import type { Scope } from "nock";
import nock from "nock";
import { describe, expect, beforeEach, afterEach, it, vi } from "vitest";
import { waitFor } from "@testing-library/react";
import { LocalStorage } from "~/utils/storage";
import Api from "~/utils/api/Api";
import * as Auth from "~/pages/auth/Auth.context";
import { mockFetchUser } from "~/testing/nocks/nock_user";
import { mockFetchTeam } from "~/testing/nocks/nock_team";
import { renderWithRouter } from "~/testing/helpers";

declare const global: {
  NavigateMock: any;
};

const { AuthContext, default: ContextProvider } = Auth;

describe("pages/auth/Auth.context", () => {
  let wrapper: RenderResult;
  let scope: Scope;
  let context: AuthContextProps;

  const mockFetchAuthProviders = ({
    status,
    response,
  }: {
    status: number;
    response: {
      github: boolean;
      gitlab: boolean;
      bitbucket: boolean;
      basicAuth?: "enabled";
    };
  }) =>
    nock(process.env.API_DOMAIN || "")
      .get(`/auth/providers`)
      .reply(status, response);

  const createWrapper = () => {
    mockFetchAuthProviders({
      status: 200,
      response: { github: true, gitlab: true, bitbucket: true },
    });

    wrapper = renderWithRouter({
      initialEntries: ["/apps/1231231?my-query=1"],
      el: () => (
        <ContextProvider>
          <AuthContext.Consumer>
            {ctx => {
              context = ctx;
              return <></>;
            }}
          </AuthContext.Consumer>
        </ContextProvider>
      ),
    });

    return wrapper;
  };

  describe("when there is a session token but it is expired", () => {
    beforeEach(() => {
      LocalStorage.set("skit_token", "my-expired-token");
      LocalStorage.set("skit_access_token", "my-access-token");
      LocalStorage.set("skit_provider", "github");

      mockFetchTeam({ response: [] });

      scope = mockFetchUser({ response: { ok: false, user: null } });
      createWrapper();
    });

    afterEach(() => {
      LocalStorage.del("skit_provider");
      LocalStorage.del("skit_access_token");
      LocalStorage.del("skit_token");
    });

    it("redirects user to the auth page", async () => {
      vi.spyOn(Api, "getAuthToken").mockReturnValue("my-expired-token");

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(global.NavigateMock).toHaveBeenCalledWith(
          "/auth?redirect=%2Fapps%2F1231231%3Fmy-query%3D1"
        );
      });
    });
  });

  describe("when there is a session token and user is logged in", () => {
    beforeEach(() => {
      LocalStorage.set("skit_token", "my-valid-token");
      LocalStorage.set("skit_access_token", "my-access-token");
      LocalStorage.set("skit_provider", "github");

      mockFetchTeam({ response: [] });

      scope = mockFetchUser({});
      createWrapper();
    });

    afterEach(() => {
      LocalStorage.del("skit_token");
    });

    it("shows the context properly", async () => {
      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(context).toEqual({
          user: {
            displayName: "stormkit",
            avatar: "https://avatars2.githubusercontent.com/u/3321893?v=4",
            email: "foo@bar",
            memberSince: 1551184200,
            id: "1",
            fullName: "Foo Bar",
            package: { id: "free" },
          },
          accounts: [
            { provider: "gitlab", url: "", displayName: "Stormkit" },
            {
              provider: "bitbucket",
              url: "https://bitbucket.org/%7B6e4d532c-e1b6-4496-90cb-f94f09af2bda%7D/",
              displayName: "stormkit",
            },
            {
              provider: "github",
              url: "https://api.github.com/users/stormkit",
              displayName: "stormkit",
            },
          ],
          authError: undefined,
          loginOauth: expect.any(Function),
          logout: expect.any(Function),
          reloadTeams: expect.any(Function),
          teams: undefined,
          providers: {
            github: true,
            gitlab: true,
            bitbucket: true,
          },
        });
      });
    });
  });
});
