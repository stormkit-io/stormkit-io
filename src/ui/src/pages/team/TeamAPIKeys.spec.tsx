import type { Scope } from "nock";
import { RenderResult, waitFor } from "@testing-library/react";
import { fireEvent, render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {
  mockFetchAPIKeys,
  mockDeleteAPIKey,
  mockGenerateAPIKey,
} from "~/testing/nocks/nock_api_keys";
import mockTeams from "~/testing/data/mock_teams";
import { mockAPIKeysResponse } from "~/testing/data";
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

      const subheader =
        "This key will allow you to interact with our API and modify all apps under this team.";

      // Header
      expect(wrapper.getByText("API Keys")).toBeTruthy();
      expect(wrapper.getByText(subheader)).toBeTruthy();

      // API Keys
      expect(wrapper.getByText("Default")).toBeTruthy();
      expect(wrapper.getByText("CI")).toBeTruthy();
    });
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

  test("developers can only create read-only keys and delete read-only ones", async () => {
    teams = mockTeams();
    const team = { ...teams[0], currentUserRole: "developer" as const };
    const keys = mockAPIKeysResponse().keys;
    keys[1].readOnly = true;

    fetchScope = mockFetchAPIKeys({ teamId: team.id, response: { keys } });
    wrapper = render(<TeamAPIKeys team={team} />);

    await waitFor(() => {
      expect(fetchScope.isDone()).toBe(true);
      expect(wrapper.getByText("CI")).toBeTruthy();
    });

    // Full-access key: no delete. Read-only key: delete available.
    expect(() => wrapper.getByLabelText("expand-8224011755")).toThrow();
    expect(wrapper.getByLabelText("expand-9868814106")).toBeTruthy();

    fireEvent.click(wrapper.getByText("New API Key"));

    await waitFor(() => {
      expect(wrapper.getByText("Generate New Key")).toBeTruthy();
    });

    const readOnlySwitch = wrapper.getByRole("switch", {
      name: "Read only",
    }) as HTMLInputElement;

    expect(readOnlySwitch.checked).toBe(true);
    expect(readOnlySwitch.disabled).toBe(true);

    const generateScope = mockGenerateAPIKey({
      name: "Dashboard",
      scope: "team",
      teamId: team.id,
      readOnly: true,
    });

    await userEvent.type(wrapper.getByLabelText("API Key name"), "Dashboard");
    fireEvent.click(wrapper.getByText("Create"));

    await waitFor(() => {
      expect(generateScope.isDone()).toBe(true);
    });
  });
});
