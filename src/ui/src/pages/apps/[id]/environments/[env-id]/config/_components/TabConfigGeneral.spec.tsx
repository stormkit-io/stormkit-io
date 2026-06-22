import { RenderResult, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, type Mock } from "vitest";
import { fireEvent, render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import { mockUpdateEnvironment } from "~/testing/nocks/nock_environment";
import TabConfigGeneral from "./TabConfigGeneral";

interface WrapperProps {
  app?: App;
  environment?: Environment;
}

describe("~/pages/apps/[id]/environments/[env-id]/config/_components/TabConfigGeneral.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;
  let setRefreshToken: Mock;

  const createWrapper = ({ app, environment }: WrapperProps) => {
    setRefreshToken = vi.fn();
    currentApp = app || mockApp();
    currentEnv = environment || mockEnvironments({ app: currentApp })[0];

    wrapper = render(
      <TabConfigGeneral
        app={currentApp}
        environment={currentEnv}
        setRefreshToken={setRefreshToken}
      />
    );
  };

  it("should have a form", () => {
    createWrapper({});

    expect(wrapper.getByText("General settings")).toBeTruthy();
    expect(wrapper.getByLabelText("Environment name")).toBeTruthy();
  });

  // Saving the general section must send only the general fields — never the
  // build config or environment variables of other sections.
  it("should submit only the general fields", async () => {
    createWrapper({});

    await userEvent.type(
      wrapper.getByLabelText("Priority deployment pattern"),
      "hotfix"
    );

    const scope = mockUpdateEnvironment({
      payload: {
        name: "production",
        branch: "master",
        autoPublish: true,
        autoDeploy: false,
        autoDeployBranches: "",
        autoDeployCommits: "",
        previewLinks: true,
        priorityPattern: "hotfix",
      },
      status: 200,
      response: { ok: true },
    });

    fireEvent.click(wrapper.getByText("Save"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(setRefreshToken).toHaveBeenCalled();
    });
  });
});
