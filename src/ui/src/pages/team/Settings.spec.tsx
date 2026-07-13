import type { RenderResult } from "@testing-library/react";
import { describe, expect, beforeEach, it, vi, type Mock } from "vitest";
import { AuthContext } from "~/pages/auth/Auth.context";
import { renderWithRouter } from "~/testing/helpers";
import mockTeams from "~/testing/data/mock_teams";
import mockUser from "~/testing/data/mock_user";
import Settings from "./Settings";

describe("~/pages/team/Settings.tsx", () => {
  let wrapper: RenderResult;
  let reloadTeams: Mock;

  interface Props {
    selectedTeam: Team;
    teams?: Team[];
  }

  const createWrapper = ({ teams, selectedTeam: team }: Props) => {
    reloadTeams = vi.fn();

    wrapper = renderWithRouter({
      path: "/:team/settings",
      initialEntries: [`/${team?.slug}/settings`],
      el: () => (
        <AuthContext.Provider value={{ user: mockUser(), teams, reloadTeams }}>
          <Settings />
        </AuthContext.Provider>
      ),
    });
  };

  describe("when has admin access", () => {
    const teams = mockTeams();
    teams[1].currentUserRole = "admin";

    beforeEach(() => {
      createWrapper({ selectedTeam: teams[1], teams });
    });

    it("should load all sections", () => {
      expect(wrapper.getByText("Team settings")).toBeTruthy();
      expect(wrapper.getByText("Team members")).toBeTruthy();
      expect(wrapper.getByText("Danger zone")).toBeTruthy();
    });
  });

  describe("when has developer access", () => {
    const teams = mockTeams();
    teams[1].currentUserRole = "developer";

    beforeEach(() => {
      createWrapper({ selectedTeam: teams[1], teams });
    });

    it("should load all sections", () => {
      expect(wrapper.getByText("Team settings")).toBeTruthy();
      expect(wrapper.getByText("Team members")).toBeTruthy();
      expect(() => wrapper.getByText("Danger zone")).toThrow();
    });
  });
});
