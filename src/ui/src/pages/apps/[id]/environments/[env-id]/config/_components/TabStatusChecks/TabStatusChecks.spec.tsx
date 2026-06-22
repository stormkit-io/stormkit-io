import { RenderResult, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach, type Mock } from "vitest";
import { fireEvent, render } from "@testing-library/react";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import { mockUpdateEnvironment } from "~/testing/nocks/nock_environment";
import TabStatusChecks from "./TabStatusChecks";

interface WrapperProps {
  app?: App;
  environment?: Environment;
  setRefreshToken?: () => void;
}

describe("~/pages/apps/[id]/environments/[env-id]/config/_components/TabStatusChecks/TabStatusChecks.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;
  let setRefreshToken: Mock;

  const createWrapper = ({ app, environment }: WrapperProps) => {
    setRefreshToken = vi.fn();
    currentApp = app || mockApp();
    currentEnv = environment || mockEnvironments({ app: currentApp })[0];

    wrapper = render(
      <TabStatusChecks
        app={currentApp}
        environment={currentEnv}
        setRefreshToken={setRefreshToken}
      />
    );
  };

  describe("empty state", () => {
    beforeEach(() => {
      createWrapper({});
    });

    it("should display an empty state message", () => {
      expect(
        wrapper.getByText(
          "You do not have any status checks for this environment."
        )
      ).toBeTruthy();
    });

    it("should have a button to add status checks", () => {
      expect(wrapper.getByText("Add status check")).toBeTruthy();
    });

    it("clicking the button should open the modal", () => {
      fireEvent.click(wrapper.getByText("Add status check"));
      expect(wrapper.getByText("Command")).toBeTruthy();
    });
  });

  describe("with pre-existing status checks", () => {
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
      createWrapper({ app, environment });
    });

    it("should contain a list of status checks", () => {
      expect(wrapper.getByText(checks[0].cmd)).toBeTruthy();
      expect(wrapper.getByText(checks[0].name!)).toBeTruthy();
      expect(wrapper.getByText(checks[1].cmd)).toBeTruthy();
      expect(wrapper.getByText(checks[1].name!)).toBeTruthy();
    });

    it("should open the modify modal", () => {
      fireEvent.click(wrapper.getAllByLabelText("expand").at(0)!);
      fireEvent.click(wrapper.getByText("Modify"));

      expect(wrapper.getByDisplayValue("npm run e2e")).toBeTruthy();
      expect(() => wrapper.getByDisplayValue("npm run test:random")).toThrow();
    });

    it("should open the delete confirm and delete the status check", async () => {
      fireEvent.click(wrapper.getAllByLabelText("expand").at(0)!);
      fireEvent.click(wrapper.getByText("Delete"));

      expect(
        wrapper.getByText("Are you sure you want to continue?")
      ).toBeTruthy();

      const scope = mockUpdateEnvironment({
        payload: { statusChecks: [checks[1]] },
        status: 200,
        response: {
          ok: true,
        },
      });

      fireEvent.click(wrapper.getByText("Yes, continue"));

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(() => wrapper.getByText("Confirm action")).toThrow();
      });
    });
  });
});
