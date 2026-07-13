import type { RenderResult } from "@testing-library/react";
import { describe, expect, it, vi, type Mock } from "vitest";
import { waitFor } from "@testing-library/react";
import { AuthContext } from "~/pages/auth/Auth.context";
import mockTeams from "~/testing/data/mock_teams";
import mockUser from "~/testing/data/mock_user";
import { mockInvitationAccept } from "~/testing/nocks/nock_team_members";
import { renderWithRouter } from "~/testing/helpers";
import InvitationAccept from "./InvitationAccept";

declare const global: {
  NavigateMock: any;
};

describe("~/pages/team/InvitationAccept.tsx", () => {
  let wrapper: RenderResult;
  let reloadTeams: Mock;

  interface Props {
    token: string;
  }

  const teams = mockTeams();

  const createWrapper = ({ token }: Props) => {
    reloadTeams = vi.fn();

    wrapper = renderWithRouter({
      path: "/invitation/accept",
      initialEntries: [`/invitation/accept?token=${token}`],
      el: () => (
        <AuthContext.Provider value={{ user: mockUser(), teams, reloadTeams }}>
          <InvitationAccept />
        </AuthContext.Provider>
      ),
    });
  };

  it("should accept the invitation and navigate to the teams page", async () => {
    const scope = mockInvitationAccept({
      token: "my-token",
      response: teams[1],
    });

    createWrapper({ token: "my-token" });

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(global.NavigateMock).toHaveBeenCalled();
    });
  });
});
