import type { RenderResult } from "@testing-library/react";
import type { TeamAuthWallApp } from "./actions";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, fireEvent, waitFor } from "@testing-library/react";
import mockTeams from "~/testing/data/mock_teams";
import {
  mockUpdateEnvAuthWall,
  mockUpdateEnvPublicPaths,
} from "~/testing/nocks/nock_team_auth_wall";
import AuthWallEnvironments from "./AuthWallEnvironments";

describe("~/pages/team/auth-wall/AuthWallEnvironments.tsx", () => {
  let wrapper: RenderResult;
  const team = mockTeams()[2];
  const onShowAllApps = vi.fn();
  const onUpdate = vi.fn();

  const apps = (): TeamAuthWallApp[] => [
    {
      id: "1",
      displayName: "storefront",
      environments: [
        { id: "11", name: "production", authWall: "" },
        {
          id: "12",
          name: "staging",
          authWall: "dev",
          publicPaths: ["/api/webhooks/*"],
        },
      ],
    },
    {
      id: "2",
      displayName: "docs",
      environments: [
        {
          id: "21",
          name: "production",
          authWall: "all",
          publicPaths: ["/robots.txt", "/api/webhooks/*"],
        },
      ],
    },
  ];

  const createWrapper = ({ appId }: { appId?: string } = {}) => {
    wrapper = render(
      <AuthWallEnvironments
        team={team}
        apps={apps()}
        loading={false}
        appId={appId}
        onShowAllApps={onShowAllApps}
        onUpdate={onUpdate}
      />,
    );
  };

  beforeEach(() => {
    onShowAllApps.mockReset();
    onUpdate.mockReset();
  });

  it("lists every app with the mode of each environment", () => {
    createWrapper();

    expect(() => wrapper.getByRole("combobox")).toThrow();

    expect(wrapper.getByText("storefront")).toBeTruthy();
    expect(wrapper.getByText("docs")).toBeTruthy();
    expect(wrapper.getByText("Off")).toBeTruthy();
    expect(wrapper.getByText("Deployment previews only")).toBeTruthy();
    expect(wrapper.getByText("All URLs, including domains")).toBeTruthy();
    expect(() => wrapper.getByText("Show all apps")).toThrow();
  });

  it("selects an environment by clicking its row", () => {
    createWrapper();

    fireEvent.click(wrapper.getByText("staging"));
    expect(wrapper.getByText("1 environment selected")).toBeTruthy();

    fireEvent.click(wrapper.getByLabelText("Select storefront staging"));
    expect(() => wrapper.getByText("1 environment selected")).toThrow();
  });

  it("selects all environments", () => {
    createWrapper();

    fireEvent.click(wrapper.getByLabelText("Select all environments"));
    expect(wrapper.getByText("3 environments selected")).toBeTruthy();

    fireEvent.click(wrapper.getByLabelText("Select all environments"));
    expect(() => wrapper.getByText("3 environments selected")).toThrow();
  });

  it("shows only the linked app, with a way back to all apps", () => {
    createWrapper({ appId: "2" });

    expect(wrapper.getByText("docs")).toBeTruthy();
    expect(() => wrapper.getByText("storefront")).toThrow();

    fireEvent.click(wrapper.getByText("Show all apps"));
    expect(onShowAllApps).toHaveBeenCalled();
  });

  it("sets the mode of the selected environments", async () => {
    const staging = mockUpdateEnvAuthWall({
      teamId: team.id,
      envId: "12",
      authWall: "all",
    });

    const docs = mockUpdateEnvAuthWall({
      teamId: team.id,
      envId: "21",
      authWall: "all",
    });

    createWrapper();

    fireEvent.click(wrapper.getByLabelText("Select storefront staging"));
    fireEvent.click(wrapper.getByLabelText("Select docs production"));

    expect(wrapper.getByText("2 environments selected")).toBeTruthy();

    fireEvent.click(wrapper.getByText("Apply"));

    await waitFor(() => {
      expect(staging.isDone()).toBe(true);
      expect(docs.isDone()).toBe(true);
      expect(
        wrapper.getByText(
          "Auth Wall of 2 environments set to: All URLs, including domains.",
        ),
      ).toBeTruthy();
    });

    expect(() => wrapper.getByText("2 environments selected")).toThrow();
    expect(onUpdate).toHaveBeenCalledTimes(1);
  });

  it("keeps the environments that failed selected", async () => {
    mockUpdateEnvAuthWall({ teamId: team.id, envId: "11", authWall: "all" });
    mockUpdateEnvAuthWall({
      teamId: team.id,
      envId: "12",
      authWall: "all",
      status: 500,
    });

    createWrapper();

    fireEvent.click(wrapper.getByLabelText("Select storefront production"));
    fireEvent.click(wrapper.getByLabelText("Select storefront staging"));
    fireEvent.click(wrapper.getByText("Apply"));

    await waitFor(() => {
      expect(
        wrapper.getByText(
          "Could not update 1 environment. It is still selected so you can retry.",
        ),
      ).toBeTruthy();
    });

    expect(wrapper.getByText("1 environment selected")).toBeTruthy();
    expect(
      (wrapper.getByLabelText("Select storefront staging") as HTMLInputElement)
        .checked,
    ).toBe(true);
  });

  it("shows the public paths of each environment", () => {
    createWrapper();

    expect(wrapper.getByText("/api/webhooks/*")).toBeTruthy();
    expect(wrapper.getByText("2 paths")).toBeTruthy();
    expect(wrapper.getByText("None")).toBeTruthy();
  });

  it("edits the public paths of an environment without selecting it", async () => {
    const scope = mockUpdateEnvPublicPaths({
      teamId: team.id,
      envId: "12",
      publicPaths: ["/api/webhooks/*", "/robots.txt"],
    });

    createWrapper();

    fireEvent.click(
      wrapper.getByLabelText("Edit public paths of storefront staging"),
    );

    expect(() => wrapper.getByText("1 environment selected")).toThrow();

    const input = wrapper.getByLabelText("Public paths") as HTMLTextAreaElement;
    expect(input.value).toBe("/api/webhooks/*");

    fireEvent.change(input, {
      target: { value: "/api/webhooks/*\n\n  /robots.txt  " },
    });

    fireEvent.click(wrapper.getByText("Save"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(
        wrapper.getByText("Public paths of storefront / staging saved."),
      ).toBeTruthy();
    });

    expect(onUpdate).toHaveBeenCalledTimes(1);
  });

  it("shows why public paths could not be saved", async () => {
    mockUpdateEnvPublicPaths({
      teamId: team.id,
      envId: "11",
      publicPaths: ["/api*"],
      status: 400,
      response: { error: 'Public path "/api*" can only use * at the end.' },
    });

    createWrapper();

    fireEvent.click(
      wrapper.getByLabelText("Edit public paths of storefront production"),
    );

    fireEvent.change(wrapper.getByLabelText("Public paths"), {
      target: { value: "/api*" },
    });

    fireEvent.click(wrapper.getByText("Save"));

    await waitFor(() => {
      expect(
        wrapper.getByText('Public path "/api*" can only use * at the end.'),
      ).toBeTruthy();
    });

    expect(onUpdate).not.toHaveBeenCalled();
  });
});
