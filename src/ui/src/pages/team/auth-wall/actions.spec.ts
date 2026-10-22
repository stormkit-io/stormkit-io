import { describe, it, expect } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { mockFetchTeamEnvs } from "~/testing/nocks/nock_team_auth_wall";
import { useFetchTeamEnvs } from "./actions";

describe("~/pages/team/auth-wall/actions.ts", () => {
  // A slow response for the previous team must not replace the current one.
  it("keeps the apps of the selected team when switching teams", async () => {
    const slow = mockFetchTeamEnvs({
      teamId: "1",
      delay: 150,
      apps: [{ id: "10", displayName: "old-team-app", environments: [] }],
    });

    const fast = mockFetchTeamEnvs({
      teamId: "2",
      apps: [{ id: "20", displayName: "new-team-app", environments: [] }],
    });

    const { result, rerender } = renderHook(
      ({ teamId }) => useFetchTeamEnvs({ teamId }),
      { initialProps: { teamId: "1" } },
    );

    rerender({ teamId: "2" });

    await waitFor(() => {
      expect(fast.isDone()).toBe(true);
      expect(result.current.apps[0]?.displayName).toBe("new-team-app");
    });

    // Let the slow response, delayed by 150ms, arrive.
    await new Promise(resolve => setTimeout(resolve, 300));
    expect(slow.isDone()).toBe(true);

    expect(result.current.apps[0]?.displayName).toBe("new-team-app");
  });
});
