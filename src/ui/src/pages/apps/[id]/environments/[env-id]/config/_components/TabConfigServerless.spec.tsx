import { RenderResult, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, type Mock } from "vitest";
import { fireEvent, render } from "@testing-library/react";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import { mockUpdateEnvironment } from "~/testing/nocks/nock_environment";
import TabConfigServerless from "./TabConfigServerless";

interface WrapperProps {
  app?: App;
  environment?: Environment;
}

describe("~/pages/apps/[id]/environments/[env-id]/config/_components/TabConfigServerless.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;
  let setRefreshToken: Mock;

  const createWrapper = ({ app, environment }: WrapperProps) => {
    setRefreshToken = vi.fn();
    currentApp = app || mockApp();
    currentEnv =
      environment ||
      ({
        ...mockEnvironments({ app: currentApp })[0],
        build: {
          ...mockEnvironments({ app: currentApp })[0].build,
          apiFolder: "/functions",
          apiPathPrefix: "/fn",
        },
      } as Environment);

    wrapper = render(
      <TabConfigServerless
        app={currentApp}
        environment={currentEnv}
        setRefreshToken={setRefreshToken}
      />
    );
  };

  it("should have a form", () => {
    createWrapper({});

    expect(wrapper.getByText("Serverless functions")).toBeTruthy();
    expect(wrapper.getByLabelText("API folder")).toBeTruthy();
  });

  // Saving serverless settings must send only the serverless fields.
  it("should submit only the serverless fields", async () => {
    createWrapper({});

    const scope = mockUpdateEnvironment({
      payload: { apiFolder: "/functions", apiPathPrefix: "/fn" },
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
