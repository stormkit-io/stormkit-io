import type { Scope } from "nock";
import { RenderResult, waitFor } from "@testing-library/react";
import { fireEvent, render } from "@testing-library/react";
import {
  mockFetchAPIKeys,
  mockDeleteAPIKey,
} from "~/testing/nocks/nock_api_keys";
import mockTeams from "~/testing/data/mock_teams";
import TeamAPIKeys from "./TeamAPIKeys";

interface WrapperProps {
  team?: Team;
}

describe("~/pages/team/TeamAPIKeys.tsx", () => {
  let wrapper: RenderResult;
  let fetchScope: Scope;
  let teams: Team[];

  const createWrapper = ({ team }: WrapperProps) => {
    teams = mockTeams();
    team = team || teams[0];

    fetchScope = mockFetchAPIKeys({ teamId: team.id });

    wrapper = render(<TeamAPIKeys team={team} />);
  };

  test("should fetch api keys", async () => {
    createWrapper({});

    await waitFor(() => {
      expect(fetchScope.isDone()).toBe(true);
    });

    const subheader =
      "This key will allow you to interact with our API and modify all apps under this team.";

    // Header
    expect(wrapper.getByText("API Keys")).toBeTruthy();
    expect(wrapper.getByText(subheader)).toBeTruthy();

    // API Keys
    expect(wrapper.getByText("Default")).toBeTruthy();
    expect(wrapper.getByText("CI")).toBeTruthy();
  });

  test("should delete api key", async () => {
    createWrapper({});

    await waitFor(() => {
      expect(wrapper.getByText("CI")).toBeTruthy();
    });

    await fireEvent.click(wrapper.getByLabelText("expand-9868814106"));
    fireEvent.click(wrapper.getByText("Delete"));

    await waitFor(() => {
      expect(wrapper.getByText("Confirm action")).toBeTruthy();
    });

    const scope = mockDeleteAPIKey({
      keyId: "9868814106",
    });

    // Should refetch api keys
    const fetchScope2 = mockFetchAPIKeys({
      teamId: teams[0].id,
      response: { keys: [] },
    });

    fireEvent.click(wrapper.getByText("Yes, continue"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(fetchScope2.isDone()).toBe(true);
      // Should close modal
      expect(() => wrapper.getByText("Confirm action")).toThrow();
      // Should no longer have API keys
      expect(() => wrapper.getByText("Default")).toThrow();
    });
  });
});
