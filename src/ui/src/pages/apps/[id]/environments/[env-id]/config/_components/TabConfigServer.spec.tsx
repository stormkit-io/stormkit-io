import { RenderResult, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, type Mock } from "vitest";
import { fireEvent, render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import { mockUpdateEnvironment } from "~/testing/nocks/nock_environment";
import TabConfigServer from "./TabConfigServer";

interface WrapperProps {
  app?: App;
  environment?: Environment;
  setRefreshToken?: () => void;
}

describe("~/pages/apps/[id]/environments/[env-id]/config/_components/TabConfigServer.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;
  let setRefreshToken: Mock;

  const createWrapper = ({ app, environment }: WrapperProps) => {
    setRefreshToken = vi.fn();
    currentApp = app || mockApp();
    currentEnv = environment || mockEnvironments({ app: currentApp })[0];

    wrapper = render(
      <TabConfigServer
        app={currentApp}
        environment={currentEnv}
        setRefreshToken={setRefreshToken}
      />
    );
  };

  it("should have a form", () => {
    createWrapper({});

    expect(wrapper.getByText("Server settings")).toBeTruthy();
    expect(
      wrapper.getByText("Configure your long-running processes.")
    ).toBeTruthy();

    const startCmdInput = wrapper.getByLabelText(
      "Start command"
    ) as HTMLInputElement;

    expect(startCmdInput.value).toBe("");
  });

  it("should submit the form", async () => {
    createWrapper({});

    await userEvent.type(
      wrapper.getByLabelText("Start command"),
      "npm run start"
    );

    const scope = mockUpdateEnvironment({
      payload: { serverCmd: "npm run start" },
      status: 200,
      response: {
        ok: true,
      },
    });

    fireEvent.click(wrapper.getByText("Save"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });
  });
});
