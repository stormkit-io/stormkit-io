import { describe, expect, it } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import nock from "nock";
import mockTeams from "~/testing/data/mock_teams";
import mockTeamMembers from "~/testing/data/mock_team_members";
import { mockFetchTeamMembers } from "~/testing/nocks/nock_team_members";
import { useFetchTeamMembers } from "./actions";

const endpoint = process.env.API_DOMAIN || "";

describe("~/pages/team/actions.ts", () => {
  describe("useFetchTeamMembers", () => {
    it("should ignore a late response for a team it no longer shows", async () => {
      const teams = mockTeams();
      const [first, second] = [teams[1], teams[2]];
      const staleMembers = mockTeamMembers({ teamId: first.id });
      const currentMembers = mockTeamMembers({ teamId: second.id });

      const staleScope = nock(endpoint)
        .get(`/team/members?teamId=${first.id}`)
        .delay(200)
        .reply(200, staleMembers);

      mockFetchTeamMembers({ teamId: second.id, response: currentMembers });

      const { result, rerender } = renderHook(
        ({ team }) => useFetchTeamMembers({ team }),
        { initialProps: { team: first } }
      );

      rerender({ team: second });

      await waitFor(() => {
        expect(result.current.teamMembers).toEqual(currentMembers);
      });

      // isDone turns true when the request is intercepted, not when the
      // delayed reply lands, so also wait out the delay.
      await waitFor(() => {
        expect(staleScope.isDone()).toBe(true);
      });

      await new Promise(resolve => setTimeout(resolve, 300));

      expect(result.current.teamMembers).toEqual(currentMembers);
      expect(result.current.loading).toBe(false);
    });
  });
});
