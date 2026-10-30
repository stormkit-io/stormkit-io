import type { RenderResult } from "@testing-library/react";
import type { TeamAuthWallApp, TeamAuthWallLogin } from "./actions";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, fireEvent, waitFor, within } from "@testing-library/react";
import mockTeams from "~/testing/data/mock_teams";
import {
  mockCreateTeamLogin,
  mockUpdateTeamLogin,
} from "~/testing/nocks/nock_team_auth_wall";
import AuthWallLoginModal from "./AuthWallLoginModal";

describe("~/pages/team/auth-wall/AuthWallLoginModal.tsx", () => {
  let wrapper: RenderResult;
  const team = mockTeams()[2];
  const onClose = vi.fn();
  const onSuccess = vi.fn();

  const apps: TeamAuthWallApp[] = [
    { id: "1", displayName: "storefront", environments: [] },
    { id: "2", displayName: "docs", environments: [] },
  ];

  const createWrapper = (
    login?: TeamAuthWallLogin,
    { magicLinkAvailable = true } = {},
  ) => {
    wrapper = render(
      <AuthWallLoginModal
        team={team}
        apps={apps}
        login={login}
        magicLinkAvailable={magicLinkAvailable}
        onClose={onClose}
        onSuccess={onSuccess}
      />,
    );
  };

  const type = (label: string, value: string) => {
    fireEvent.change(wrapper.getByLabelText(label), { target: { value } });
  };

  const selectApp = async (name: string) => {
    fireEvent.mouseDown(wrapper.getByRole("combobox"));
    fireEvent.click(
      within(await waitFor(() => wrapper.getByRole("listbox"))).getByText(name),
    );
    fireEvent.keyDown(wrapper.getByRole("listbox"), { key: "Escape" });
  };

  beforeEach(() => {
    onClose.mockReset();
    onSuccess.mockReset();
  });

  it("creates a login for the selected apps", async () => {
    const scope = mockCreateTeamLogin({
      body: {
        teamId: team.id,
        email: "jane@example.org",
        password: "jane-password",
        metadata: "acme",
        magicLink: false,
        allApps: false,
        appIds: ["2"],
      },
    });

    createWrapper();
    type("Email address", "jane@example.org");
    type("Password", "jane-password");
    type("Metadata", "acme");
    await selectApp("docs");

    fireEvent.click(wrapper.getByText("Create"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(onSuccess).toHaveBeenCalled();
      expect(onClose).toHaveBeenCalled();
    });
  });

  it("requires an app, or all apps", () => {
    createWrapper();
    type("Email address", "jane@example.org");
    type("Password", "jane-password");

    fireEvent.click(wrapper.getByText("Create"));

    expect(
      wrapper.getByText("Select at least one app, or all apps."),
    ).toBeTruthy();
  });

  it("shows the error returned by the API", async () => {
    const scope = mockCreateTeamLogin({
      body: {
        teamId: team.id,
        email: "jane@example.org",
        password: "jane-password",
        metadata: "",
        magicLink: false,
        allApps: true,
        appIds: [],
      },
      status: 409,
      response: {
        error: "A login with the same email already exists in this team.",
      },
    });

    createWrapper();
    type("Email address", "jane@example.org");
    type("Password", "jane-password");
    fireEvent.click(
      wrapper.getByLabelText("All apps, including apps created later"),
    );
    fireEvent.click(wrapper.getByText("Create"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(
        wrapper.getByText(
          "A login with the same email already exists in this team.",
        ),
      ).toBeTruthy();
    });

    expect(onClose).not.toHaveBeenCalled();
  });

  it("keeps the password when editing without a new one", async () => {
    const scope = mockUpdateTeamLogin({
      body: {
        teamId: team.id,
        id: "7",
        metadata: "vip",
        magicLink: false,
        allApps: true,
        appIds: [],
      },
    });

    createWrapper({
      id: "7",
      email: "jane@example.org",
      metadata: "",
      allApps: false,
      appIds: ["1"],
    });

    expect(
      (wrapper.getByLabelText("Email address") as HTMLInputElement).disabled,
    ).toBe(true);

    type("Metadata", "vip");
    fireEvent.click(
      wrapper.getByLabelText("All apps, including apps created later"),
    );
    fireEvent.click(wrapper.getByText("Save"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(onSuccess).toHaveBeenCalled();
    });
  });

  it("creates a login that signs in with magic links only", async () => {
    const scope = mockCreateTeamLogin({
      body: {
        teamId: team.id,
        email: "jane@example.org",
        password: "",
        metadata: "",
        magicLink: true,
        allApps: true,
        appIds: [],
      },
    });

    createWrapper();
    type("Email address", "jane@example.org");
    fireEvent.click(
      wrapper.getByLabelText("Allow signing in with a link sent by email"),
    );
    fireEvent.click(
      wrapper.getByLabelText("All apps, including apps created later"),
    );
    fireEvent.click(wrapper.getByText("Create"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(onSuccess).toHaveBeenCalled();
    });
  });

  it("requires a password or magic links", async () => {
    createWrapper();
    type("Email address", "jane@example.org");
    fireEvent.click(
      wrapper.getByLabelText("All apps, including apps created later"),
    );
    fireEvent.click(wrapper.getByText("Create"));

    await waitFor(() => {
      expect(
        wrapper.getByText("Set a password, allow magic links, or both."),
      ).toBeTruthy();
    });
  });

  it("removes the password of a login that allows magic links", async () => {
    const scope = mockUpdateTeamLogin({
      body: {
        teamId: team.id,
        id: "7",
        metadata: "",
        magicLink: true,
        allApps: false,
        appIds: ["1"],
        removePassword: true,
      },
    });

    createWrapper({
      id: "7",
      email: "jane@example.org",
      metadata: "",
      hasPassword: true,
      magicLink: false,
      allApps: false,
      appIds: ["1"],
    });

    expect(() =>
      wrapper.getByLabelText(
        "Remove the password, so the login signs in with links only",
      ),
    ).toThrow();

    fireEvent.click(
      wrapper.getByLabelText("Allow signing in with a link sent by email"),
    );
    fireEvent.click(
      wrapper.getByLabelText(
        "Remove the password, so the login signs in with links only",
      ),
    );

    expect(
      (wrapper.getByLabelText("Password") as HTMLInputElement).disabled,
    ).toBe(true);

    fireEvent.click(wrapper.getByText("Save"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(onSuccess).toHaveBeenCalled();
    });
  });

  it("does not offer magic links without a mailer", () => {
    createWrapper(undefined, { magicLinkAvailable: false });

    const checkbox = wrapper.getByLabelText(
      "Allow signing in with a link sent by email",
    ) as HTMLInputElement;

    expect(checkbox.disabled).toBe(true);
    expect(
      wrapper.getByText(
        "Configure a mailer under Admin > Mailer to use magic links.",
      ),
    ).toBeTruthy();
  });

  it("lets a login keep or drop magic links without a mailer", () => {
    createWrapper(
      {
        id: "7",
        email: "jane@example.org",
        metadata: "",
        magicLink: true,
        hasPassword: true,
        allApps: true,
        appIds: [],
      },
      { magicLinkAvailable: false },
    );

    const checkbox = wrapper.getByLabelText(
      "Allow signing in with a link sent by email",
    ) as HTMLInputElement;

    expect(checkbox.disabled).toBe(false);
    expect(checkbox.checked).toBe(true);
  });
});
