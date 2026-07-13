import type { RenderResult } from "@testing-library/react";
import { describe, expect, beforeEach, it, vi } from "vitest";
import { waitFor, fireEvent, render } from "@testing-library/react";
import mockApp from "~/testing/data/mock_app";
import mockTeams from "~/testing/data/mock_teams";
import { mockMigrateApp } from "~/testing/nocks/nock_team";
import FormMigrateApp from "./FormMigrateApp";

interface Props {
  app: App;
}

describe("~/pages/apps/[id]/settings/_components/FormMigrateApp", () => {
  let currentApp: App;
  let wrapper: RenderResult;
  const teams = mockTeams();

  const createWrapper = ({ app }: Props) => {
    wrapper = render(<FormMigrateApp app={app} teams={teams} />);
  };

  beforeEach(() => {
    currentApp = mockApp();
    createWrapper({ app: currentApp });
  });

  it("displays title and subtitle correctly", () => {
    expect(wrapper.getByText("Migrate app")).toBeTruthy();
    expect(
      wrapper.getByText(
        "Move this application to a different team. At least 'Admin' role in destination team is required."
      )
    );
  });

  it("developer roles should not appear in the list", () => {
    fireEvent.mouseDown(wrapper.getByText(/Personal/));
    expect(() => wrapper.getByText(teams[1].name)).toThrow();
  });

  it("clicking Move application should call the confirm action", async () => {
    fireEvent.mouseDown(wrapper.getByText(/Personal/));
    fireEvent.click(wrapper.getByText(teams[2].name));

    fireEvent.click(wrapper.getByText("Move application"));

    const modal = wrapper.getByText(/This will migrate the app/);

    expect(modal).toBeTruthy();

    fireEvent.change(wrapper.getByLabelText("Confirmation"), {
      target: { value: "migrate" },
    });

    const scope = mockMigrateApp({
      appId: currentApp.id,
      teamId: teams[2].id,
      response: { ok: true },
    });

    await fireEvent.click(wrapper.getByText("Yes, continue"));

    Object.defineProperty(window, "location", {
      value: {
        reload: vi.fn(),
      },
    });

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(window.location.reload).toHaveBeenCalled();
    });
  });
});
