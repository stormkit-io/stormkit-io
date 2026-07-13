import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, RenderResult } from "@testing-library/react";
import DeploymentLogs from "./DeploymentLogs";

interface Props {
  logs: Log[];
  isRunning: boolean;
}

vi.mock("~/utils/helpers/deployments", () => ({
  formattedDate: () => "21.09.2022 - 21:30",
}));

describe("~/apps/[id]/environments/[env-id]/deployments/DeploymentLogs.tsx", () => {
  let wrapper: RenderResult;

  const createWrapper = ({ logs, isRunning }: Props) => {
    wrapper = render(<DeploymentLogs logs={logs} isRunning={isRunning} />);
  };

  const logs: Log[] = [
    {
      title: "npm run build",
      message: "Nuxt CLI v3.0.0-rc.8",
      status: true,
      payload: "",
      duration: 10,
    },
  ];

  it("should show the log duration", () => {
    createWrapper({ logs, isRunning: false });
    expect(wrapper.getByText("10s")).toBeTruthy();
  });

  it("should display the logs when expanded", () => {
    createWrapper({ logs, isRunning: false });

    // First logs are collapsed
    expect(() => wrapper.getByText("Nuxt CLI v3.0.0-rc.8")).toThrow();

    // Then we expand
    fireEvent.click(wrapper.getByText("npm run build"));

    // Then logs are shown
    expect(wrapper.getByText("Nuxt CLI v3.0.0-rc.8")).toBeTruthy();
  });

  it("should display the logs by default when running", () => {
    createWrapper({ logs, isRunning: true });

    // Then logs are shown
    expect(wrapper.getByText("Nuxt CLI v3.0.0-rc.8")).toBeTruthy();
  });
});
