import { RenderResult, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, type Mock } from "vitest";
import { fireEvent, render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import {
  mockUpdateEnvironment,
  mockRevealEnvVars,
} from "~/testing/nocks/nock_environment";
import TabConfigEnvVars from "./TabConfigEnvVars";

interface WrapperProps {
  app?: App;
  environment?: Environment;
  setRefreshToken?: () => void;
}

describe("~/pages/apps/[id]/environments/[env-id]/config/_components/TabConfigEnvVars.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;
  let setRefreshToken: Mock;

  const createWrapper = ({ app, environment }: WrapperProps) => {
    setRefreshToken = vi.fn();
    currentApp = app || mockApp();
    currentEnv = environment || mockEnvironments({ app: currentApp })[0];

    wrapper = render(
      <TabConfigEnvVars
        app={currentApp}
        environment={currentEnv}
        setRefreshToken={setRefreshToken}
      />
    );
  };

  it("default state", () => {
    currentApp = mockApp();
    currentEnv = mockEnvironments({ app: currentApp })[0];
    currentEnv.build.vars = {};

    createWrapper({ environment: currentEnv });

    const header = "Environment variables";
    const subheader =
      "These variables will be available to build time, status checks and serverless runtime.";

    // Header
    expect(wrapper.getByText(header)).toBeTruthy();
    expect(wrapper.getByText(subheader)).toBeTruthy();

    // Table header
    expect(wrapper.getByText("Name")).toBeTruthy();
    expect(wrapper.getByText("Value")).toBeTruthy();

    // Table row
    expect(wrapper.getByPlaceholderText("NODE_ENV")).toBeTruthy();
    expect(wrapper.getByPlaceholderText("production")).toBeTruthy();

    // Buttons
    expect(wrapper.getByText("Add Row")).toBeTruthy();
    expect(wrapper.getByText("Modify as a string")).toBeTruthy();
    expect(wrapper.getByText("Save")).toBeTruthy();
  });

  it("with pre-populated environment variables", () => {
    currentApp = mockApp();
    currentEnv = mockEnvironments({ app: currentApp })[0];
    currentEnv.build.vars = {
      NODE_ENV: "production",
      STORMKIT: "true",
      TZ: "UTC+3",
    };

    createWrapper({ environment: currentEnv });

    // Table row
    expect(wrapper.getByDisplayValue("NODE_ENV")).toBeTruthy();
    expect(wrapper.getByDisplayValue("production")).toBeTruthy();

    expect(wrapper.getByDisplayValue("STORMKIT")).toBeTruthy();
    expect(wrapper.getByDisplayValue("true")).toBeTruthy();

    expect(wrapper.getByDisplayValue("TZ")).toBeTruthy();
    expect(wrapper.getByDisplayValue("UTC+3")).toBeTruthy();
  });

  it("updating environment variables", async () => {
    currentApp = mockApp();
    currentEnv = mockEnvironments({ app: currentApp })[0];
    currentEnv.build.vars = {};

    createWrapper({ environment: currentEnv });

    // No pre-configured variables, so editing is unlocked without revealing.
    await userEvent.type(
      wrapper.getByPlaceholderText("NODE_ENV"),
      "env_var_1_key"
    );

    await userEvent.type(
      wrapper.getByPlaceholderText("production"),
      "env_var_1_val"
    );

    const scope = mockUpdateEnvironment({
      payload: { envVars: { env_var_1_key: "env_var_1_val" } },
      status: 200,
      response: { ok: true },
    });

    fireEvent.click(wrapper.getByText("Save"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(setRefreshToken).toHaveBeenCalled();
    });
  });

  it("hides the reveal button when there are no variables", () => {
    currentApp = mockApp();
    currentEnv = mockEnvironments({ app: currentApp })[0];
    currentEnv.build.vars = {};

    createWrapper({ environment: currentEnv });

    expect(wrapper.queryByText("Reveal values")).toBeNull();

    const save = wrapper.getByText("Save").closest("button") as HTMLButtonElement;
    expect(save.disabled).toBe(false);
  });

  it("shows the reveal button when variables exist", () => {
    currentApp = mockApp();
    currentEnv = mockEnvironments({ app: currentApp })[0];
    currentEnv.build.vars = { EXISTING: "" };

    createWrapper({ environment: currentEnv });

    expect(wrapper.getByText("Reveal values")).toBeTruthy();

    const save = wrapper.getByText("Save").closest("button") as HTMLButtonElement;
    expect(save.disabled).toBe(true);

    // Editing affordances stay locked until the values are revealed, so the
    // user cannot type entries that would be discarded on reveal.
    const addRow = wrapper
      .getByText("Add Row")
      .closest("button") as HTMLButtonElement;
    expect(addRow.disabled).toBe(true);

    const modify = wrapper
      .getByText("Modify as a string")
      .closest("button") as HTMLButtonElement;
    expect(modify.disabled).toBe(true);
  });

  it("shows an error when reveal fails", async () => {
    currentApp = mockApp();
    currentEnv = mockEnvironments({ app: currentApp })[0];
    currentEnv.build.vars = { EXISTING: "" };

    createWrapper({ environment: currentEnv });

    const revealScope = mockRevealEnvVars({
      envId: currentEnv.id!,
      response: {},
      status: 403,
    });

    fireEvent.click(wrapper.getByText("Reveal values"));

    await waitFor(() => {
      expect(revealScope.isDone()).toBe(true);
      expect(wrapper.getByText(/Something went wrong/)).toBeTruthy();
    });
  });

  it("resetting environment variables", async () => {
    currentApp = mockApp();
    currentEnv = mockEnvironments({ app: currentApp })[0];
    currentEnv.build.vars = {};

    createWrapper({ environment: currentEnv });

    await userEvent.type(
      wrapper.getByPlaceholderText("NODE_ENV"),
      "env_var_1_key"
    );

    await userEvent.type(
      wrapper.getByPlaceholderText("production"),
      "env_var_1_val"
    );

    const scope = mockUpdateEnvironment({
      payload: { envVars: { env_var_1_key: "env_var_1_val" } },
      status: 200,
      response: { ok: true },
    });

    fireEvent.click(wrapper.getByText("Cancel"));

    await waitFor(() => {
      expect(wrapper.getByPlaceholderText("NODE_ENV")).toBeTruthy();
      expect(scope.isDone()).toBe(false);
      expect(setRefreshToken).not.toHaveBeenCalled();
    });
  });
});
