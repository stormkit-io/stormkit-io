import type { Scope } from "nock";
import type { RenderResult } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { render, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { AuthContext } from "~/pages/auth/Auth.context";
import mockDeployments from "~/testing/data/mock_deployments_v2";
import mockTeams from "~/testing/data/mock_teams";
import mockUser from "~/testing/data/mock_user";
import { mockFetchDeployments } from "~/testing/nocks/nock_deployments_v2";
import Deployments from "./Deployments";

interface Props {
  deployments?: DeploymentV2[];
}

describe("~/pages/my/deployments/Deployments.tsx", () => {
  let currentDeploys: DeploymentV2[];
  let currentTeam: Team;
  let scope: Scope;
  let wrapper: RenderResult;

  const createWrapper = async ({ deployments }: Props) => {
    const teams = mockTeams();
    currentTeam = teams[0];
    currentDeploys = deployments || mockDeployments();

    scope = mockFetchDeployments({
      teamId: currentTeam.id,
      response: { deployments: currentDeploys },
    });

    wrapper = render(
      <MemoryRouter>
        <AuthContext.Provider value={{ user: mockUser(), teams }}>
          <Deployments />
        </AuthContext.Provider>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });
  };

  it("it displays correct header and subheader", async () => {
    await createWrapper({});
    await waitFor(() => {
      expect(wrapper.getByText("Team Deployments")).toBeTruthy();
      expect(
        wrapper.getByText(
          "Display all of your team's deployments in a single view"
        )
      ).toBeTruthy();
    });
  });

  it("it displays a list of deployments", async () => {
    await createWrapper({});
    await waitFor(() => {
      expect(wrapper.getByText("chore: update packages")).toBeTruthy();
      expect(wrapper.getByText("fix: image size")).toBeTruthy();
      expect(wrapper.getByText("by Joe Doe")).toBeTruthy();
      expect(wrapper.getByText("by Sally Doe")).toBeTruthy();
    });
  });

  it("it displays an empty page when there are no deployments", async () => {
    await createWrapper({ deployments: [] });
    await waitFor(() => {
      expect(wrapper.getByText(/It's quite empty in here./)).toBeTruthy();
      expect(wrapper.getByText(/Go back to your/)).toBeTruthy();
      expect(wrapper.getByText("Apps").getAttribute("href")).toBe("/");
      expect(
        wrapper.getByText(/to start deploying your website./)
      ).toBeTruthy();
    });
  });
});
