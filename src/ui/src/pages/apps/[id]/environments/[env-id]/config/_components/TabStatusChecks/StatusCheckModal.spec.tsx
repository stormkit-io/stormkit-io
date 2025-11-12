import { describe, expect, it, vi, beforeEach, type Mock } from "vitest";
import { RenderResult, waitFor } from "@testing-library/react";
import { fireEvent, render } from "@testing-library/react";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import userEvent from "@testing-library/user-event";
import { mockUpdateEnvironment } from "~/testing/nocks/nock_environment";
import StatusCheckModal from "./StatusCheckModal";

interface WrapperProps {
  app?: App;
  environment?: Environment;
  statusCheckIndex: number;
}

describe("~/pages/apps/[id]/environments/[env-id]/config/_components/TabStatusChecks/StatusCheckModal.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;
  let setRefreshToken: Mock;
  let onClose: Mock;

  const createWrapper = ({
    app,
    environment,
    statusCheckIndex,
  }: WrapperProps) => {
    setRefreshToken = vi.fn();
    onClose = vi.fn();
    currentApp = app || mockApp();
    currentEnv = environment || mockEnvironments({ app: currentApp })[0];

    wrapper = render(
      <StatusCheckModal
        app={currentApp}
        env={currentEnv}
        setRefreshToken={setRefreshToken}
        statusCheckIndex={statusCheckIndex}
        onClose={onClose}
      />
    );
  };

  describe("create mode", () => {
    beforeEach(() => {
      createWrapper({ statusCheckIndex: -1 });
    });

    it("should display labels", () => {
      expect(wrapper.getByText("Command")).toBeTruthy();
      expect(wrapper.getByText("Name")).toBeTruthy();
      expect(wrapper.getByText("Description")).toBeTruthy();
    });

    it("should not allow for empty command inputs", () => {
      fireEvent.click(wrapper.getByText("Save"));
      expect(wrapper.getByText("Command is a required field.")).toBeTruthy();
    });

    it("should submit the form and create a status check", async () => {
      await userEvent.type(wrapper.getByLabelText("Command"), "npm run test");
      await userEvent.type(wrapper.getByLabelText("Name"), "Run e2e tests");
      await userEvent.type(wrapper.getByLabelText("Description"), "desc");

      const scope = mockUpdateEnvironment({
        environment: {
          ...currentEnv,
          build: {
            ...currentEnv.build,
            statusChecks: [
              {
                cmd: "npm run test",
                name: "Run e2e tests",
                description: "desc",
              },
            ],
          },
        },
        status: 200,
        response: {
          ok: true,
        },
      });

      fireEvent.click(wrapper.getByText("Save"));

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(setRefreshToken).toHaveBeenCalled();
        expect(onClose).toHaveBeenCalled();
      });
    });
  });

  describe("edit mode", () => {
    const app = mockApp();
    const environment = mockEnvironments({ app })[0];

    environment.build.statusChecks = [
      {
        name: "Run e2e tests",
        cmd: "npm run e2e",
        description: "My description",
      },
      {
        name: "Run random tests",
        cmd: "npm run test:random",
        description: "My other description",
      },
    ];

    const checks = environment.build.statusChecks!;

    beforeEach(() => {
      createWrapper({ app, environment, statusCheckIndex: 1 });
    });

    it("should display labels", () => {
      expect(wrapper.getByDisplayValue(checks[1].cmd)).toBeTruthy();
      expect(wrapper.getByDisplayValue(checks[1].name!)).toBeTruthy();
      expect(wrapper.getByDisplayValue(checks[1].description!)).toBeTruthy();
    });

    it("should submit the form and create a status check", async () => {
      await userEvent.type(wrapper.getByLabelText("Command"), ":2");
      await userEvent.type(wrapper.getByLabelText("Name"), " - 2");
      await userEvent.type(wrapper.getByLabelText("Description"), " - 2");

      const scope = mockUpdateEnvironment({
        environment: {
          ...currentEnv,
          build: {
            ...currentEnv.build,
            statusChecks: [
              checks[0],
              {
                cmd: "npm run test:random:2",
                name: "Run random tests - 2",
                description: "My other description - 2",
              },
            ],
          },
        },
        status: 200,
        response: {
          ok: true,
        },
      });

      fireEvent.click(wrapper.getByText("Save"));

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(setRefreshToken).toHaveBeenCalled();
        expect(onClose).toHaveBeenCalled();
      });
    });
  });
});
