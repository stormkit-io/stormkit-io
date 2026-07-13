import type { RenderResult } from "@testing-library/react";
import { describe, expect, beforeEach, it, vi, type Mock } from "vitest";
import { fireEvent, waitFor } from "@testing-library/react";
import { AuthContext } from "~/pages/auth/Auth.context";
import mockTeams from "~/testing/data/mock_teams";
import mockUser from "~/testing/data/mock_user";
import { mockInviteTeamMember } from "~/testing/nocks/nock_team_members";
import { renderWithRouter } from "~/testing/helpers";
import InviteMemberModal from "./InviteMemberModal";

describe("~/pages/team/InviteMemberModal.tsx", () => {
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
      path: "/:team/settings",
      initialEntries: [`/${team?.slug}/settings`],
      el: () => (
        <AuthContext.Provider value={{ user: mockUser(), teams, reloadTeams }}>
          <InviteMemberModal teamId={teams[1].id} onClose={onClose} />
        </AuthContext.Provider>
      ),
    });
  };

  beforeEach(() => {
    createWrapper({ team: teams[1] });
  });

  it("should display title and subtitle", () => {
    expect(wrapper.getByText("Invite team member")).toBeTruthy();
    expect(
      wrapper.getByText(
        "Team members will be able to see and modify all projects in the team."
      )
    ).toBeTruthy();
  });

  it("should create an invitation token for the invited user", async () => {
    const scope = mockInviteTeamMember({
      teamId: teams[1].id,
      email: "jamie@stormkit.io",
      role: "admin",
      response: { token: "my-token" },
    });

    fireEvent.change(wrapper.getByLabelText("Email"), {
      target: { value: "jamie@stormkit.io" },
    });

    fireEvent.click(wrapper.getByText("Invite"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);

      expect(
        wrapper.getByText(
          "Share the following link with your colleague to invite them to the team."
        )
      ).toBeTruthy();

      expect(
        wrapper.getByDisplayValue(
          "http://localhost:3000/invitation/accept?token=my-token"
        )
      ).toBeTruthy();
    });
  });
});
