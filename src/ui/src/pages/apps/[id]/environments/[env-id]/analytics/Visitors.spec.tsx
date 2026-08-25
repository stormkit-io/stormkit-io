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
      />,
    );
  };

  it("should include correct texts", async () => {
    createWrapper({});

    expect(
      wrapper.getByText("Bots are excluded from these statistics"),
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
      expect(wrapper.getByTestId("area-chart").innerHTML).toEqual(
        JSON.stringify([
          { name: "2024-01-14", total: 20, unique: 10 },
          { name: "2024-01-13", total: 50, unique: 25 },
          { name: "2023-12-19", total: 46, unique: 19 },
          { name: "2023-11-04", total: 32, unique: 22 },
          { name: "2023-07-02", total: 12, unique: 12 },
        ]),
      );
    });
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
    // Local noon so the calendar date is the same in every timezone.
    vi.useFakeTimers({
      now: new Date(2024, 0, 14, 12, 0, 0).getTime(),
      toFake: ["Date"],
    });

    createWrapper({
      ts: "7d",
    });

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText("Visitors")).toBeTruthy();
      expect(wrapper.getAllByText(/Total/).at(0)).toBeTruthy();
      // 2024-01-13 (50) only. Today (2024-01-14, 20) is excluded because the
      // daily aggregation has not run for it yet.
      expect(wrapper.getByText("50")).toBeTruthy();
      expect(wrapper.getByText(/visits in the last/)).toBeTruthy();
      expect(wrapper.getByText(/7 days/)).toBeTruthy();
    });

    vi.useRealTimers();
  });

  it("should end the 7d window at yesterday, not today", async () => {
    vi.useFakeTimers({
      now: new Date(2024, 0, 14, 12, 0, 0).getTime(),
      toFake: ["Date"],
    });

    createWrapper({ ts: "7d" });

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByTestId("area-chart").innerHTML).toEqual(
        JSON.stringify([
          { name: "2024-01-07", total: 0, unique: 0 },
          { name: "2024-01-08", total: 0, unique: 0 },
          { name: "2024-01-09", total: 0, unique: 0 },
          { name: "2024-01-10", total: 0, unique: 0 },
          { name: "2024-01-11", total: 0, unique: 0 },
          { name: "2024-01-12", total: 0, unique: 0 },
          { name: "2024-01-13", total: 50, unique: 25 },
        ]),
      );
    });

    vi.useRealTimers();
  });

  it("should end the 30d window at yesterday, not today", async () => {
    vi.useFakeTimers({
      now: new Date(2024, 0, 14, 12, 0, 0).getTime(),
      toFake: ["Date"],
    });

    createWrapper({ ts: "30d" });

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });

    const series = JSON.parse(wrapper.getByTestId("area-chart").innerHTML);

    expect(series).toHaveLength(30);
    expect(series[0].name).toBe("2023-12-15");
    expect(series[series.length - 1]).toEqual({
      name: "2024-01-13",
      total: 50,
      unique: 25,
    });
    expect(series.find((s: { name: string }) => s.name === "2024-01-14")).toBe(
      undefined,
    );

    vi.useRealTimers();
  });

  it("should not drop today for the 24h span", async () => {
    createWrapper({ ts: "24h" });

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      // 24h renders whatever keys the API returns, untouched.
      expect(wrapper.getByTestId("area-chart").innerHTML).toEqual(
        JSON.stringify([
          { name: "2024-01-14", total: 20, unique: 10 },
          { name: "2024-01-13", total: 50, unique: 25 },
          { name: "2023-12-19", total: 46, unique: 19 },
          { name: "2023-11-04", total: 32, unique: 22 },
          { name: "2023-07-02", total: 12, unique: 12 },
        ]),
      );
    });
  });
});
