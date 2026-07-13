import { RenderResult, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, type Mock, beforeEach } from "vitest";
import { fireEvent, render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AuthContext } from "~/pages/auth/Auth.context";
import { RootContext } from "~/pages/Root.context";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import mockUser from "~/testing/data/mock_user";
import { mockUpdateEnvironment } from "~/testing/nocks/nock_environment";
import TabConfigBuild from "./TabConfigBuild";

interface WrapperProps {
  app?: App;
  environment?: Environment;
  setRefreshToken?: () => void;
  user?: User;
  edition?: InstanceDetails["stormkit"]["edition"];
}

describe("~/pages/apps/[id]/environments/[env-id]/config/_components/TabConfigBuild.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;
  let setRefreshToken: Mock;

  const createWrapper = ({
    app,
    environment,
    user,
    edition = "self-hosted",
  }: WrapperProps) => {
    setRefreshToken = vi.fn();
    currentApp = app || mockApp();
    currentEnv = environment || mockEnvironments({ app: currentApp })[0];

    wrapper = render(
      <RootContext.Provider
        value={{
          mode: "dark",
          setMode: () => {},
          details: { stormkit: { edition, apiCommit: "", apiVersion: "" } },
        }}
      >
        <AuthContext.Provider value={{ user: user || mockUser() }}>
          <TabConfigBuild
            app={currentApp}
            environment={currentEnv}
            setRefreshToken={setRefreshToken}
          />
        </AuthContext.Provider>
      </RootContext.Provider>
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
      expect(wrapper.getByLabelText("Cache directories")).toBeTruthy();
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

      fireEvent.change(wrapper.getByLabelText("Cache directories"), {
        target: { value: ".next/cache\nnode_modules" },
      });

      const scope = mockUpdateEnvironment({
        payload: {
          installCmd: "go get .",
          buildCmd: "go build .",
          distFolder: "./dist",
          workDir: "./",
          cacheDirs: [".next/cache", "node_modules"],
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

  describe("build cache gating", () => {
    it("is editable for premium users on cloud", () => {
      createWrapper({ user: mockUser({ packageId: "premium" }), edition: "cloud" });

      const field = wrapper.getByLabelText("Cache directories");

      expect(field.hasAttribute("disabled")).toBe(false);
    });

    it("is disabled with an upsell for free users on cloud", () => {
      createWrapper({ user: mockUser({ packageId: "free" }), edition: "cloud" });

      const field = wrapper.getByLabelText("Cache directories");

      expect(field.hasAttribute("disabled")).toBe(true);
      expect(wrapper.getByText("Upgrade to enterprise")).toBeTruthy();
    });

    it("is editable on self-hosted regardless of package", () => {
      createWrapper({ user: mockUser({ packageId: "free" }), edition: "self-hosted" });

      const field = wrapper.getByLabelText("Cache directories");

      expect(field.hasAttribute("disabled")).toBe(false);
    });
  });

  it("pre-configured state", () => {
    currentApp = mockApp();
    currentEnv = mockEnvironments({ app: currentApp })[0];
    currentEnv.build.installCmd = "go get .";
    currentEnv.build.buildCmd = "go build main.go";
    currentEnv.build.distFolder = "./";
    currentEnv.build.workDir = "./root";
    currentEnv.build.cacheDirs = [".next/cache", "node_modules"];

    createWrapper({ environment: currentEnv });

    expect(wrapper.getByDisplayValue("go get .")).toBeTruthy();
    expect(wrapper.getByDisplayValue("go build main.go")).toBeTruthy();
    expect(wrapper.getByDisplayValue("./")).toBeTruthy();
    expect(wrapper.getByDisplayValue("./root")).toBeTruthy();
    const cacheField = wrapper.getByLabelText(
      "Cache directories"
    ) as HTMLTextAreaElement;

    expect(cacheField.value).toBe(".next/cache\nnode_modules");
  });
});
