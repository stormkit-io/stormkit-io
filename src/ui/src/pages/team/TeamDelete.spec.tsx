import type { RenderResult } from "@testing-library/react";
import { describe, expect, beforeEach, it, vi, type Mock } from "vitest";
import { fireEvent, waitFor } from "@testing-library/react";
import mockTeams from "~/testing/data/mock_teams";
import { mockRemoveTeam } from "~/testing/nocks/nock_team";
import { renderWithRouter } from "~/testing/helpers";
import TeamDelete from "./TeamDelete";

describe("~/pages/team/TeamDelete.tsx", () => {
  let wrapper: RenderResult;
  let reloadTeams: Mock;

  interface Props {
    team: Team;
  }

  const createWrapper = ({ team }: Props) => {
    reloadTeams = vi.fn();

    wrapper = renderWithRouter({
      path: "/:team/settings",
      initialEntries: [`/${team?.slug}/settings`],
      el: () => <TeamDelete team={team} reloadTeams={reloadTeams} />,
    });
  };

  describe("when has admin access", () => {
    const teams = mockTeams();
    teams[1].currentUserRole = "owner";

    beforeEach(() => {
      createWrapper({ team: teams[1] });
    });

    it("should display title and subtitle", () => {
      expect(wrapper.getByText("Danger zone")).toBeTruthy();
      expect(
        wrapper.getByText(
          "Permanently delete your Team and all of its contents from the Stormkit platform. This action cannot be undone — proceed with caution."
        )
      ).toBeTruthy();
    });

    it("should remove the team", async () => {
      const scope = mockRemoveTeam({
        teamId: teams[1].id,
      });

      fireEvent.click(wrapper.getByText("Delete"));

      fireEvent.change(wrapper.getByLabelText("Confirmation"), {
        target: { value: "Permanently delete my team" },
      });

      fireEvent.click(wrapper.getByText("Yes, continue"));

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(reloadTeams).toHaveBeenCalled();
      });
    });
  });

  describe("when has developer access", () => {
    const teams = mockTeams();
    teams[1].currentUserRole = "developer";

    beforeEach(() => {
      createWrapper({ team: teams[1] });
    });

    it("should not render anyting", () => {
      expect(wrapper.container.innerHTML).toBe("");
    });
  });
});
