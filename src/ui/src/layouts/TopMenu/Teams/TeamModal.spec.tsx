import type { RenderResult } from "@testing-library/react";
import { describe, expect, beforeEach, it, vi, type Mock } from "vitest";
import { fireEvent, waitFor } from "@testing-library/react";
import mockTeams from "~/testing/data/mock_teams";
import { mockCreateTeam } from "~/testing/nocks/nock_team";
import TeamModal from "./TeamModal";
import { renderWithRouter } from "~/testing/helpers";

declare const global: {
  NavigateMock: any;
};

describe("~/layouts/TopMenu/Teams/TeamModal.tsx", () => {
  let wrapper: RenderResult;
  let onClose: Mock;
  let reloadTeams: Mock;

  interface Props {
    team?: Team;
  }

  const teams = mockTeams();

  const createWrapper = ({ team }: Props) => {
    onClose = vi.fn();
    reloadTeams = vi.fn();

    wrapper = renderWithRouter({
      path: "/:team?",
      initialEntries: [`/${team?.slug}`],
      el: () => <TeamModal onClose={onClose} reload={reloadTeams} />,
    });
  };

  beforeEach(() => {
    createWrapper({});
  });

  it("should display header", () => {
    expect(wrapper.getByText("New Team")).toBeTruthy();
  });

  it("should display subheader", () => {
    expect(
      wrapper.getByText("Use teams to collaborate with other team members.")
    ).toBeTruthy();
  });

  it("should not allow creating empty teams", () => {
    fireEvent.click(wrapper.getByText("Create"));
    expect(wrapper.getByText("Team name is a required field.")).toBeTruthy();
  });

  it("should create a team", async () => {
    const team = { ...teams[2], name: "My team", slug: "my-team" };
    const scope = mockCreateTeam({ name: team.name, response: team });

    fireEvent.change(wrapper.getByLabelText("Team name"), {
      target: { value: team.name },
    });

    fireEvent.click(wrapper.getByText("Create"));

    await waitFor(() => {
      scope.isDone();
      expect(onClose).toHaveBeenCalled();
      expect(reloadTeams).toHaveBeenCalled();
      expect(global.NavigateMock).toHaveBeenCalledWith(`/${team.slug}`, {
        replace: true,
      });
    });
  });
});
