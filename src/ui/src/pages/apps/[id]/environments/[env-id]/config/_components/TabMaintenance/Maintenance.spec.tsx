import type { RenderResult } from "@testing-library/react";
import { describe, it, expect, beforeEach } from "vitest";
import { render, waitFor, fireEvent } from "@testing-library/react";
import mockApp from "~/testing/data/mock_app";
import mockEnvironment from "~/testing/data/mock_environment";
import * as actions from "~/testing/nocks/nock_maintenance";
import Maintenance from "./Maintenance";

const { mockFetchMaintenanceConfig, mockUpdateMaintenanceConfig } = actions;

describe("~/pages/apps/[id]/environments/[env-id]/config/_components/TabMaintenance/Maintenance.tsx", () => {
  let wrapper: RenderResult;
  let app: App;
  let env: Environment;

  beforeEach(() => {
    app = mockApp();
    env = mockEnvironment({ app });
  });

  const createWrapper = () => {
    wrapper = render(<Maintenance app={app} environment={env} />);
  };

  it.each`
    config  | expectedStatus | checked
    ${""}   | ${"Disabled"}  | ${false}
    ${"on"} | ${"Enabled"}   | ${true}
  `(
    "reflects the API response in the toggle and status",
    async ({ config, expectedStatus, checked }) => {
      const scope = mockFetchMaintenanceConfig({
        appId: app.id,
        envId: env.id!,
        response: { maintenance: config },
      });

      createWrapper();
      expect(wrapper.getByTestId("card-loading")).toBeTruthy();

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(() => wrapper.getByTestId("card-loading")).toThrow();
        expect(wrapper.getByText(expectedStatus)).toBeTruthy();

        const toggle = wrapper.getByLabelText(
          "Turn on maintenance mode"
        ) as HTMLInputElement;

        expect(toggle.checked).toBe(checked);
      });
    }
  );

  it("turns on maintenance mode", async () => {
    const fetchScope = mockFetchMaintenanceConfig({
      appId: app.id,
      envId: env.id!,
      response: { maintenance: "" },
    });

    createWrapper();

    await waitFor(() => {
      expect(fetchScope.isDone()).toBe(true);
    });

    const updateScope = mockUpdateMaintenanceConfig({
      appId: app.id,
      envId: env.id!,
      maintenance: "on",
    });

    // The refetch after a successful update
    mockFetchMaintenanceConfig({
      appId: app.id,
      envId: env.id!,
      response: { maintenance: "on" },
    });

    fireEvent.click(wrapper.getByLabelText("Turn on maintenance mode"));
    fireEvent.click(wrapper.getByText("Save"));

    await waitFor(() => {
      expect(updateScope.isDone()).toBe(true);
      expect(
        wrapper.getByText("Maintenance mode configuration updated successfully.")
      ).toBeTruthy();
    });
  });

  it("turns off maintenance mode", async () => {
    const fetchScope = mockFetchMaintenanceConfig({
      appId: app.id,
      envId: env.id!,
      response: { maintenance: "on" },
    });

    createWrapper();

    await waitFor(() => {
      expect(fetchScope.isDone()).toBe(true);
    });

    const updateScope = mockUpdateMaintenanceConfig({
      appId: app.id,
      envId: env.id!,
      maintenance: "",
    });

    mockFetchMaintenanceConfig({
      appId: app.id,
      envId: env.id!,
      response: { maintenance: "" },
    });

    fireEvent.click(wrapper.getByLabelText("Turn on maintenance mode"));
    fireEvent.click(wrapper.getByText("Save"));

    await waitFor(() => {
      expect(updateScope.isDone()).toBe(true);
      expect(
        wrapper.getByText("Maintenance mode configuration updated successfully.")
      ).toBeTruthy();
    });
  });
});
