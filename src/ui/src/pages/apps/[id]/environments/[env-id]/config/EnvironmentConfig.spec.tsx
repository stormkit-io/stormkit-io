import { RenderResult, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach } from "vitest";
import { MemoryRouter } from "react-router";
import { fireEvent, render } from "@testing-library/react";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import { AppContext } from "~/pages/apps/[id]/App.context";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import EnvironmentConfig from "./EnvironmentConfig";

interface WrapperProps {
  app?: App;
  environments?: Environment[];
  hash?: string;
  setRefreshToken?: () => void;
}

describe("~/pages/apps/[id]/environments/[env-id]/config/EnvironmentConfig.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;
  let currentEnvs: Environment[];

  const createWrapper = ({
    app,
    environments,
    hash,
    setRefreshToken = () => {},
  }: WrapperProps) => {
    currentApp = app || mockApp();
    currentEnvs = environments || mockEnvironments({ app: currentApp });
    currentEnv = currentEnvs[0];

    wrapper = render(
      <MemoryRouter initialEntries={[`/${hash}`]}>
        <AppContext.Provider
          value={{
            app: currentApp,
            environments: currentEnvs,
            setRefreshToken,
          }}
        >
          <EnvironmentContext.Provider value={{ environment: currentEnv }}>
            <EnvironmentConfig />
          </EnvironmentContext.Provider>
        </AppContext.Provider>
      </MemoryRouter>,
    );
  };

  it("should default to the #general tab", () => {
    createWrapper({});

    expect(wrapper.getByText("General settings")).toBeTruthy();

    expect(wrapper.getByLabelText("Environment name")).toBeTruthy();
    expect(wrapper.getAllByDisplayValue(currentEnv.name)).toHaveLength(3); // env vars (2) and env name

    expect(wrapper.getByLabelText("Branch")).toBeTruthy();
    expect(wrapper.getByDisplayValue(currentEnv.branch)).toBeTruthy();

    expect(wrapper.getByLabelText("Auto publish")).toBeTruthy();
    expect(wrapper.getByLabelText("Auto deploy")).toBeTruthy();
  });

  it("should contain tabs to switch views", () => {
    createWrapper({});
    expect(wrapper.getByText("General")).toBeTruthy();
    expect(wrapper.getByText("Build")).toBeTruthy();
    expect(wrapper.getAllByText("Environment variables").at(0)).toBeTruthy();
    expect(wrapper.getByText("API Keys")).toBeTruthy();
    expect(wrapper.getByText("Status checks")).toBeTruthy();
    expect(wrapper.getByText("Serverless functions")).toBeTruthy();
    expect(wrapper.getByText("Headers")).toBeTruthy();
    expect(wrapper.getByText("Redirects")).toBeTruthy();
    expect(wrapper.getByText("Auth wall")).toBeTruthy();
    expect(wrapper.getByText("Domains")).toBeTruthy();
  });

  it.each`
    hash           | expectedString
    ${""}          | ${"Use these settings to configure your environment details."}
    ${""}          | ${"These variables will be available to build time, status checks and serverless runtime."}
    ${""}          | ${"Use these settings to configure your build options."}
    ${"#authwall"} | ${"Limit access to your deployments with an authentication wall."}
    ${"#api-keys"} | ${"This key will allow you to interact with our API and modify this environment."}
  `(
    "should load different tab based on hash: $hash",
    ({ hash, expectedString }) => {
      createWrapper({ hash });
      expect(wrapper.getByText(expectedString)).toBeTruthy();
    },
  );

  it("should switch between tabs", async () => {
    createWrapper({});

    fireEvent.click(wrapper.getAllByText("Environment variables").at(0)!);

    await waitFor(() => {
      expect(
        wrapper.getByTestId("env-config-nav").getAttribute("data-selected"),
      ).toBe("#env-vars");
    });
  });

  describe("when it is a bare app", () => {
    beforeEach(() => {
      createWrapper({ app: { ...currentApp, isBare: true } });
    });

    it("should not display the build tab", () => {
      expect(() => wrapper.getByText("Build")).toThrow();
    });

    it("should not display the Status Checks tab", () => {
      expect(() => wrapper.getByText("Status checks")).toThrow();
    });
  });
});
