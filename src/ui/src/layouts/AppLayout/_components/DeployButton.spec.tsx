import { describe, expect, afterEach, beforeEach, it } from "vitest";
import { fireEvent, type RenderResult } from "@testing-library/react";
import mockApp from "~/testing/data/mock_app";
import mockEnv from "~/testing/data/mock_environment";
import DeployButton from "./DeployButton";
import { renderWithRouter } from "~/testing/helpers";

interface Props {
  app: App;
  env: Environment;
}

describe("~/layouts/AppLayout/_components/DeployButton.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;

  const createWrapper = ({ app, env }: Props) => {
    wrapper = renderWithRouter({
      el: () => (
        <DeployButton app={app} environments={[env]} selectedEnvId={env.id!} />
      ),
    });
  };

  afterEach(() => {
    wrapper.unmount();
  });

  describe("without query parameter", () => {
    beforeEach(() => {
      currentApp = mockApp();
      currentEnv = mockEnv({ app: currentApp });
      createWrapper({ app: currentApp, env: currentEnv });
    });

    it("mounts the button properly", () => {
      expect(wrapper.getByText("Deploy")).toBeTruthy();
      expect(() => wrapper.getByText("Start a deployment")).toThrow();
    });

    it("displays the modal when button is clicked", () => {
      fireEvent.click(wrapper.getByText("Deploy"));
      expect(wrapper.getByText("Start a deployment")).toBeTruthy();
    });
  });
});
