import { RenderResult, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, type Mock, beforeEach } from "vitest";
import { fireEvent, render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import { mockUpdateEnvironment } from "~/testing/nocks/nock_environment";
import TabConfigBuild from "./TabConfigBuild";

interface WrapperProps {
  app?: App;
  environment?: Environment;
  setRefreshToken?: () => void;
}

describe("~/pages/apps/[id]/environments/[env-id]/config/_components/TabConfigBuild.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;
  let setRefreshToken: Mock;

  const createWrapper = ({ app, environment }: WrapperProps) => {
    setRefreshToken = vi.fn();
    currentApp = app || mockApp();
    currentEnv = environment || mockEnvironments({ app: currentApp })[0];

    wrapper = render(
      <TabConfigBuild
        app={currentApp}
        environment={currentEnv}
        setRefreshToken={setRefreshToken}
      />
    );
  };

  describe("default state", () => {
    beforeEach(() => {
      currentApp = mockApp();
      currentEnv = mockEnvironments({ app: currentApp })[0];
      currentEnv.build.installCmd = "";
      currentEnv.build.buildCmd = "";
      currentEnv.build.distFolder = "";
      currentEnv.build.vars = {};

      createWrapper({ environment: currentEnv });
    });

    it("default state", () => {
      const header = "Build settings";
      const subheader = "Use these settings to configure your build options.";

      // Header
      expect(wrapper.getByText(header)).toBeTruthy();
      expect(wrapper.getByText(subheader)).toBeTruthy();

      expect(wrapper.getByLabelText("Install command")).toBeTruthy();
      expect(wrapper.getByLabelText("Build command")).toBeTruthy();
      expect(wrapper.getByLabelText("Output folder")).toBeTruthy();
      expect(wrapper.getByLabelText("Build root")).toBeTruthy();
    });

    it("should update the environment", async () => {
      await userEvent.type(
        wrapper.getByLabelText("Install command"),
        "go get ."
      );

      await userEvent.type(
        wrapper.getByLabelText("Build command"),
        "go build ."
      );

      await userEvent.type(wrapper.getByLabelText("Output folder"), "dist");

      const scope = mockUpdateEnvironment({
        payload: {
          installCmd: "go get .",
          buildCmd: "go build .",
          distFolder: "./dist",
          workDir: "./",
          skipUnchangedBuildRoot: false,
          watchPaths: [],
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

  describe("build root path filtering", () => {
    beforeEach(() => {
      currentApp = mockApp();
      currentEnv = mockEnvironments({ app: currentApp })[0];
      currentEnv.build.installCmd = "";
      currentEnv.build.buildCmd = "";
      currentEnv.build.distFolder = "";
      currentEnv.build.vars = {};

      createWrapper({ environment: currentEnv });
    });

    it("keeps typed watch paths when the filter is toggled off and on", async () => {
      await userEvent.click(
        wrapper.getByLabelText("Deploy only when the build root changes")
      );

      fireEvent.change(wrapper.getByLabelText("Watch paths"), {
        target: { value: "packages/ui" },
      });

      // Toggling the switch must not discard what the user typed.
      await userEvent.click(
        wrapper.getByLabelText("Deploy only when the build root changes")
      );
      await userEvent.click(
        wrapper.getByLabelText("Deploy only when the build root changes")
      );

      expect(
        (wrapper.getByLabelText("Watch paths") as HTMLTextAreaElement).value
      ).toBe("packages/ui");
    });

    it("sends the filter and the watch paths", async () => {
      await userEvent.click(
        wrapper.getByLabelText("Deploy only when the build root changes")
      );

      fireEvent.change(wrapper.getByLabelText("Watch paths"), {
        target: { value: "packages/ui\npackages/config" },
      });

      const scope = mockUpdateEnvironment({
        payload: {
          installCmd: "",
          buildCmd: "",
          distFolder: "./",
          workDir: "./",
          skipUnchangedBuildRoot: true,
          watchPaths: ["packages/ui", "packages/config"],
        },
        status: 200,
        response: { ok: true },
      });

      fireEvent.click(wrapper.getByText("Save"));

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
      });
    });
  });

  it("pre-configured state", () => {
    currentApp = mockApp();
    currentEnv = mockEnvironments({ app: currentApp })[0];
    currentEnv.build.installCmd = "go get .";
    currentEnv.build.buildCmd = "go build main.go";
    currentEnv.build.distFolder = "./";
    currentEnv.build.workDir = "./root";

    createWrapper({ environment: currentEnv });

    expect(wrapper.getByDisplayValue("go get .")).toBeTruthy();
    expect(wrapper.getByDisplayValue("go build main.go")).toBeTruthy();
    expect(wrapper.getByDisplayValue("./")).toBeTruthy();
    expect(wrapper.getByDisplayValue("./root")).toBeTruthy();
  });
});
