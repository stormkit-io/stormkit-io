import type { RenderResult } from "@testing-library/react";
import type { TeamAuthWallApp, TeamAuthWallLogin } from "./actions";
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent, waitFor } from "@testing-library/react";
import mockTeams from "~/testing/data/mock_teams";
import {
  mockFetchTeamLogins,
  mockDeleteTeamLogins,
} from "~/testing/nocks/nock_team_auth_wall";
import AuthWallLogins from "./AuthWallLogins";

vi.mock("~/utils/helpers/date", () => ({
  formatDate: () => "21.09.2022 - 21:30",
}));

describe("~/pages/team/auth-wall/AuthWallLogins.tsx", () => {
  let wrapper: RenderResult;
  const team = mockTeams()[2];

  const apps: TeamAuthWallApp[] = [
    { id: "1", displayName: "storefront", environments: [] },
    { id: "2", displayName: "docs", environments: [] },
  ];

  const login = (props: Partial<TeamAuthWallLogin>): TeamAuthWallLogin => ({
    id: "1",
    email: "jane@example.org",
    metadata: "",
    allApps: false,
    appIds: [],
    ...props,
  });

  const createWrapper = ({ appsError }: { appsError?: string } = {}) => {
    wrapper = render(
      <AuthWallLogins team={team} apps={apps} appsError={appsError} />,
    );
  };

  it("shows an empty state", async () => {
    const scope = mockFetchTeamLogins({ teamId: team.id });
    createWrapper();

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(
        wrapper.getByText(
          "This team has no logins yet. Click the button above to add one.",
        ),
      ).toBeTruthy();
    });
  });

  it("lists logins with the apps they open", async () => {
    mockFetchTeamLogins({
      teamId: team.id,
      logins: [
        login({
          id: "1",
          email: "jane@example.org",
          appIds: ["1", "2"],
          lastLogin: 1743904881,
          hasPassword: true,
          magicLink: true,
        }),
        login({
          id: "2",
          email: "joe@example.org",
          allApps: true,
          hasPassword: false,
          magicLink: true,
        }),
      ],
    });

    createWrapper();

    await waitFor(() => {
      expect(wrapper.getByText("jane@example.org")).toBeTruthy();
    });

    expect(wrapper.getByText("storefront, docs")).toBeTruthy();
    expect(wrapper.getByText("All apps")).toBeTruthy();
    expect(wrapper.getByText("Password, Magic link")).toBeTruthy();
    expect(wrapper.getByText("Magic link")).toBeTruthy();
    expect(wrapper.getByText("21.09.2022 - 21:30")).toBeTruthy();
    expect(wrapper.getByText("Never")).toBeTruthy();
  });

  it("loads the next page", async () => {
    mockFetchTeamLogins({
      teamId: team.id,
      logins: [login({ id: "1", email: "jane@example.org" })],
      hasNextPage: true,
    });

    createWrapper();

    await waitFor(() => {
      expect(wrapper.getByText("Load more")).toBeTruthy();
    });

    const scope = mockFetchTeamLogins({
      teamId: team.id,
      from: 1,
      logins: [login({ id: "2", email: "joe@example.org" })],
    });

    fireEvent.click(wrapper.getByText("Load more"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText("joe@example.org")).toBeTruthy();
    });

    expect(wrapper.getByText("jane@example.org")).toBeTruthy();
    expect(() => wrapper.getByText("Load more")).toThrow();
  });

  it("removes the selected logins", async () => {
    mockFetchTeamLogins({
      teamId: team.id,
      logins: [
        login({ id: "1", email: "jane@example.org" }),
        login({ id: "2", email: "joe@example.org" }),
      ],
    });

    createWrapper();

    await waitFor(() => {
      expect(wrapper.getByText("joe@example.org")).toBeTruthy();
    });

    fireEvent.click(wrapper.getByLabelText("joe@example.org"));
    fireEvent.click(wrapper.getByText("Remove selected"));

    const deleteScope = mockDeleteTeamLogins({
      teamId: team.id,
      loginIds: ["2"],
    });
    const fetchScope = mockFetchTeamLogins({
      teamId: team.id,
      logins: [login({ id: "1", email: "jane@example.org" })],
    });

    fireEvent.click(wrapper.getByText("Yes, continue"));

    await waitFor(() => {
      expect(deleteScope.isDone()).toBe(true);
      expect(fetchScope.isDone()).toBe(true);
      expect(() => wrapper.getByText("joe@example.org")).toThrow();
    });
  });

  it("blocks new logins when the apps could not be loaded", async () => {
    mockFetchTeamLogins({ teamId: team.id });
    createWrapper({
      appsError: "Something went wrong while fetching the environments.",
    });

    await waitFor(() => {
      expect(
        wrapper.getByText(
          "Something went wrong while fetching the environments.",
        ),
      ).toBeTruthy();
    });

    expect(
      (wrapper.getByLabelText("New login") as HTMLButtonElement).disabled,
    ).toBe(true);
  });

  it("warns about link-only logins when there is no mailer", async () => {
    mockFetchTeamLogins({
      teamId: team.id,
      magicLinkAvailable: false,
      logins: [
        login({ id: "1", hasPassword: false, magicLink: true }),
        login({ id: "2", email: "joe@example.org", hasPassword: true }),
      ],
    });

    createWrapper();

    await waitFor(() => {
      expect(
        wrapper.getByText(/1 login signs in with magic links only/),
      ).toBeTruthy();
    });
  });

  it("does not warn when the instance can send links", async () => {
    mockFetchTeamLogins({
      teamId: team.id,
      logins: [login({ id: "1", hasPassword: false, magicLink: true })],
    });

    createWrapper();

    await waitFor(() => {
      expect(wrapper.getByText("jane@example.org")).toBeTruthy();
    });

    expect(() => wrapper.getByText(/signs in with magic links only/)).toThrow();
  });
});
