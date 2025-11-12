import type { TimeSpan } from "./index.d";
import type { RenderResult } from "@testing-library/react";
import type { Scope } from "nock";
import { waitFor, render, fireEvent } from "@testing-library/react";
import { describe, expect, it, vi, Mock } from "vitest";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import mockVisitors from "~/testing/data/mock_analytics_visitors";
import mockDomain from "~/testing/data/mock_domain";
import { mockFetchVisitors } from "~/testing/nocks/nock_analytics";
import Visitors from "./Visitors";

interface WrapperProps {
  app?: App;
  ts?: TimeSpan;
}

describe("~/pages/apps/[id]/environments/[env-id]/analytics/Visitors.tsx", () => {
  let wrapper: RenderResult;
  let scope: Scope;
  let currentApp: App;
  let currentEnv: Environment;
  let currentEnvs: Environment[];
  let onTimeSpanChange: Mock;
  const data = mockVisitors();

  const createWrapper = ({ app, ts = "24h" }: WrapperProps) => {
    const domain = mockDomain();
    currentApp = app || mockApp();
    currentEnvs = mockEnvironments({ app: currentApp });
    currentEnv = currentEnvs[0];
    onTimeSpanChange = vi.fn();

    scope = mockFetchVisitors({
      unique: "false",
      ts,
      envId: currentEnv.id,
      domainId: domain.id,
      response: data,
    });

    wrapper = render(
      <Visitors
        environment={currentEnv}
        onTimeSpanChange={onTimeSpanChange}
        domain={domain}
        ts={ts}
      />
    );
  };

  it("should include correct texts", async () => {
    createWrapper({});

    expect(
      wrapper.getByText("Bots are excluded from these statistics")
    ).toBeTruthy();

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText("Visitors")).toBeTruthy();
      expect(wrapper.getAllByText(/Total/).at(0)).toBeTruthy();
      expect(wrapper.getByText("160")).toBeTruthy();
      expect(wrapper.getByText(/visits in the last/)).toBeTruthy();
      expect(wrapper.getByText("24 hours")).toBeTruthy();
    });
  });

  it("should fetch visitors from the api", async () => {
    createWrapper({});

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });

    expect(wrapper.getByTestId("area-chart").innerHTML).toEqual(
      JSON.stringify([
        { name: "2024-01-14", total: 20, unique: 10 },
        { name: "2024-01-13", total: 50, unique: 25 },
        { name: "2023-12-19", total: 46, unique: 19 },
        { name: "2023-11-04", total: 32, unique: 22 },
        { name: "2023-07-02", total: 12, unique: 12 },
      ])
    );
  });

  it("should emit event when time span changes", async () => {
    createWrapper({});

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });

    fireEvent.click(wrapper.getByText("7 Days"));

    await waitFor(() => {
      expect(onTimeSpanChange).toHaveBeenCalledWith("7d");
    });
  });

  it("should fetch with specified time span", async () => {
    vi.useFakeTimers({
      now: new Date("2024-01-14").getTime(),
      toFake: ["Date"],
    });

    createWrapper({
      ts: "7d",
    });

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText("Visitors")).toBeTruthy();
      expect(wrapper.getAllByText(/Total/).at(0)).toBeTruthy();
      expect(wrapper.getByText("70")).toBeTruthy();
      expect(wrapper.getByText(/visits in the last/)).toBeTruthy();
      expect(wrapper.getByText(/7 days/)).toBeTruthy();
    });

    vi.useRealTimers();
  });
});
