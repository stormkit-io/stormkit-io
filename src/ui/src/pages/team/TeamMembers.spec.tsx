import type { RenderResult } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach, type Mock } from "vitest";
import { fireEvent, waitFor } from "@testing-library/react";
import mockTeams from "~/testing/data/mock_teams";
import mockUser from "~/testing/data/mock_user";
import mockTeamMembers from "~/testing/data/mock_team_members";
import {
  mockFetchTeamMembers,
  mockRemoveTeamMember,
} from "~/testing/nocks/nock_team_members";
import TeamMembers from "./TeamMembers";
import { renderWithRouter } from "~/testing/helpers";

describe("~/pages/team/TeamMembers.tsx", () => {
  let wrapper: RenderResult;
  let reloadTeams: Mock;
  let members: TeamMember[];

  interface Props {
    team: Team;
  }

  const createWrapper = ({ team }: Props) => {
    reloadTeams = vi.fn();

    wrapper = renderWithRouter({
      path: "/:team/settings",
      initialEntries: [`/${team?.slug}/settings`],
      el: () => (
        <TeamMembers team={team} reloadTeams={reloadTeams} user={mockUser()} />
      ),
    });
  };

  describe("when has admin access", () => {
    const teams = mockTeams();
    teams[1].currentUserRole = "admin";

    beforeEach(() => {
      members = mockTeamMembers({ teamId: teams[1].id });
      mockFetchTeamMembers({ teamId: teams[1].id, response: members });
      createWrapper({ team: teams[1] });
    });

    it("should display title and subtitle", () => {
      expect(wrapper.getByText("Team members")).toBeTruthy();
      expect(
        wrapper.getByText(
          "Invite team members to collaborate on your projects."
        )
      ).toBeTruthy();
    });

    it("should list team members", async () => {
      await waitFor(() => {
        expect(wrapper.getByText(members[0].displayName)).toBeTruthy();
        expect(wrapper.getByText(members[1].displayName)).toBeTruthy();
        expect(wrapper.getByText(members[2].displayName)).toBeTruthy();
        expect(wrapper.getByText(members[0].email)).toBeTruthy();
        expect(wrapper.getByText(members[1].email)).toBeTruthy();
        expect(wrapper.getByText(members[2].email)).toBeTruthy();
      });
    });

    it("invite member button should open the modal", () => {
      fireEvent.click(wrapper.getByText("Invite member"));
      expect(wrapper.getByText("Invite team member")).toBeTruthy();
    });

    it("should remove team member", async () => {
      let menu;

      const scope = mockRemoveTeamMember({
        teamId: members[1].teamId,
        memberId: members[1].id,
      });

      await waitFor(() => {
        menu = wrapper.getByLabelText(`Member ${members[1].id} menu`);
      });

      fireEvent.click(menu!);
      fireEvent.click(wrapper.getByText("Remove"));
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
      members = mockTeamMembers({ teamId: teams[1].id });
      mockFetchTeamMembers({ teamId: teams[1].id, response: members });
      createWrapper({ team: teams[1] });
    });

    it("should display title and subtitle", () => {
      expect(wrapper.getByText("Team members")).toBeTruthy();
      expect(
        wrapper.getByText(
          "Invite team members to collaborate on your projects."
        )
      ).toBeTruthy();
    });

    it("should list team members", async () => {
      await waitFor(() => {
        expect(wrapper.getByText(members[0].displayName)).toBeTruthy();
        expect(wrapper.getByText(members[1].displayName)).toBeTruthy();
        expect(wrapper.getByText(members[2].displayName)).toBeTruthy();
        expect(wrapper.getByText(members[0].email)).toBeTruthy();
        expect(wrapper.getByText(members[1].email)).toBeTruthy();
        expect(wrapper.getByText(members[2].email)).toBeTruthy();
      });
    });

    it("should not display the invite member button", () => {
      expect(() => wrapper.getByText("Invite member")).toThrow();
    });
  });
});
