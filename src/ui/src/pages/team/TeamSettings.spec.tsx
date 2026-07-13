import type { RenderResult } from "@testing-library/react";
import { describe, expect, beforeEach, it, vi, type Mock } from "vitest";
import { fireEvent, waitFor } from "@testing-library/react";
import mockTeams from "~/testing/data/mock_teams";
import { mockUpdateTeam } from "~/testing/nocks/nock_team";
import { renderWithRouter } from "~/testing/helpers";
import TeamSettings from "./TeamSettings";

declare const global: {
  NavigateMock: any;
};

describe("~/pages/team/TeamSettings.tsx", () => {
  let wrapper: RenderResult;
  let reloadTeams: Mock;

  interface Props {
    team: Team;
  }

  const createWrapper = ({ team }: Props) => {
    reloadTeams = vi.fn();

    wrapper = renderWithRouter({
      path: "/:teamId/settings",
      initialEntries: [`/${team?.slug}/settings`],
      el: () => <TeamSettings team={team} reloadTeams={reloadTeams} />,
    });
  };

  describe("when has admin rights", () => {
    const teams = mockTeams();
    teams[1].currentUserRole = "owner";

    beforeEach(() => {
      createWrapper({ team: teams[1] });
    });

    it("should display title and subtitle", () => {
      expect(wrapper.getByText("Team settings")).toBeTruthy();
      expect(
        wrapper.getByText("Only Owners and Admins can update team settings.")
      ).toBeTruthy();
    });

    it("should update a team", async () => {
      const team = { ...teams[1], name: "My new name", slug: "my-new-name" };
      const scope = mockUpdateTeam({
        name: team.name,
        teamId: team.id,
        response: team,
      });

      fireEvent.change(wrapper.getByLabelText("Team name"), {
        target: { value: team.name },
      });

      fireEvent.click(wrapper.getByText("Update"));

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(reloadTeams).toHaveBeenCalled();
        expect(global.NavigateMock).toHaveBeenCalledWith(
          `/${team.id}/settings`,
          {
            replace: true,
          }
        );
      });
    });
  });

  describe("when has developer rights", () => {
    const teams = mockTeams();
    teams[1].currentUserRole = "developer";

    beforeEach(() => {
      createWrapper({ team: teams[1] });
    });

    it("should display title and subtitle", () => {
      expect(wrapper.getByText("Team settings")).toBeTruthy();
      expect(
        wrapper.getByText("Only Owners and Admins can update team settings.")
      ).toBeTruthy();
    });

    it("should not have an update button", () => {
      expect(() => wrapper.getByText("Update")).toThrow();
    });
  });
});
