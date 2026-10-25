import type { TimeSpan } from "./index.d";
import type { RenderResult } from "@testing-library/react";
import type { Scope } from "nock";
import { waitFor, render, fireEvent } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi, Mock } from "vitest";
import { trackFetches } from "~/testing/helpers";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import mockVisitors from "~/testing/data/mock_analytics_visitors";
import mockReport from "~/testing/data/mock_analytics_report";
import mockDomain from "~/testing/data/mock_domain";
import {
  mockFetchVisitors,
  mockFetchAnalyticsReport,
} from "~/testing/nocks/nock_analytics";
import Visitors, { formatDuration, tooltipLabel } from "./Visitors";

interface WrapperProps {
  ts?: TimeSpan;
}

// The day spans end yesterday in UTC, so they hold complete days only.
const yesterdayUTC = () => {
  const date = new Date();
  date.setUTCDate(date.getUTCDate() - 1);

  return date.toISOString().slice(0, 10);
};

describe("~/pages/apps/[id]/environments/[env-id]/analytics/Visitors.tsx", () => {
  let wrapper: RenderResult;
  let scope: Scope;
  let currentEnv: Environment;
  let onTimeSpanChange: Mock;

  const reportQuery = (domain: Domain, ts: TimeSpan) => ({
    envId: currentEnv.id!,
    domainId: domain.id,
    metric: "visitors",
    span: ts,
    to: yesterdayUTC(),
    compare: "previous",
  });

  const createWrapper = ({ ts = "24h" }: WrapperProps) => {
    const domain = mockDomain();
    currentEnv = mockEnvironments({ app: mockApp() })[0];
    onTimeSpanChange = vi.fn();

    scope =
      ts === "24h"
        ? mockFetchVisitors({
            unique: "false",
            ts,
            envId: currentEnv.id,
            domainId: domain.id,
            response: mockVisitors(),
          })
        : mockFetchAnalyticsReport({
            query: reportQuery(domain, ts),
            response: mockReport(),
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

  describe("last 24 hours", () => {
    it("charts the hourly page views and visitors", async () => {
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

    it("sums the page views without a comparison", async () => {
      createWrapper({});

      await waitFor(() => {
        expect(wrapper.getByText("160")).toBeTruthy();
        expect(wrapper.getByText(/in the last 24 hours/)).toBeTruthy();
        expect(wrapper.queryByText(/%/)).toBeNull();
      });

      // Hourly unique visitors do not add up to unique visitors, so the
      // header does not total them.
      expect(wrapper.queryByText(/visitors in the last/)).toBeNull();
      expect(
        wrapper.getByText(/a person active in two hours counts twice/),
      ).toBeTruthy();
    });
  });

  describe("daily spans", () => {
    it("charts the report's series", async () => {
      createWrapper({ ts: "7d" });

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(wrapper.getByTestId("area-chart").innerHTML).toEqual(
          JSON.stringify([
            { name: "2024-01-12", total: 300, unique: 100 },
            { name: "2024-01-13", total: 500, unique: 150 },
            { name: "2024-01-14", total: 400, unique: 150 },
          ]),
        );
      });
    });

    it("shows the totals with their change against the previous period", async () => {
      createWrapper({ ts: "7d" });

      await waitFor(() => {
        expect(wrapper.getByText("1,200")).toBeTruthy();
        expect(wrapper.getByText("+20%")).toBeTruthy();
        expect(wrapper.getByText("400")).toBeTruthy();
        expect(wrapper.getByText("-20%")).toBeTruthy();
        expect(wrapper.getByText("1m 35s")).toBeTruthy();
        expect(wrapper.getByText("+0%")).toBeTruthy();
        expect(wrapper.getByText(/in the last 7 days/)).toBeTruthy();
      });
    });

    it.each<[TimeSpan, string]>([
      ["30d", "30 days"],
      ["90d", "90 days"],
      ["365d", "year"],
    ])("reads the %s report", async (ts, text) => {
      createWrapper({ ts });

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(
          wrapper.getByText(new RegExp(`in the last ${text}`)),
        ).toBeTruthy();
      });
    });
  });

  it.each<[string, TimeSpan]>([
    ["7 Days", "7d"],
    ["90 Days", "90d"],
    ["1 Year", "365d"],
  ])("emits %s when its span is picked", async (label, ts) => {
    createWrapper({});

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });

    fireEvent.click(wrapper.getByText(label));

    await waitFor(() => {
      expect(onTimeSpanChange).toHaveBeenCalledWith(ts);
    });
  });

  it("explains how daily visitors are counted", async () => {
    createWrapper({ ts: "7d" });

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });

    expect(
      wrapper.getByText(/a person who visits on two days counts twice/),
    ).toBeTruthy();
  });

  // A slow response for an earlier span must not replace the current one.
  it("ignores the response of a span that is no longer selected", async () => {
    const domain = mockDomain();
    currentEnv = mockEnvironments({ app: mockApp() })[0];

    const slow = mockFetchAnalyticsReport({
      query: reportQuery(domain, "365d"),
      response: {
        ...mockReport(),
        series: [{ date: "2023-01-01", pageViews: 9, visitors: 9 }],
      },
      delay: 300,
    });

    const fast = mockFetchAnalyticsReport({
      query: reportQuery(domain, "30d"),
      response: mockReport(),
    });

    const props = {
      environment: currentEnv,
      onTimeSpanChange: vi.fn(),
      domain,
    };
    const view = render(<Visitors {...props} ts="365d" />);

    view.rerender(<Visitors {...props} ts="30d" />);

    await waitFor(() => {
      expect(fast.isDone()).toBe(true);
      expect(slow.isDone()).toBe(true);
    });

    await new Promise(r => setTimeout(r, 350));

    expect(JSON.parse(view.getByTestId("area-chart").innerHTML)).toHaveLength(
      3,
    );
  });

  describe("switching domains", () => {
    const props = () => ({
      environment: currentEnv,
      onTimeSpanChange: vi.fn(),
    });

    const hourly = (domain: Domain, total: number, delay = 0) =>
      mockFetchVisitors({
        unique: "false",
        ts: "24h",
        envId: currentEnv.id,
        domainId: domain.id,
        response: { "2024-01-14 13:00": { total, unique: 1 } },
        delay,
      });

    const chart = () => JSON.parse(wrapper.getByTestId("area-chart").innerHTML);

    let fetches: ReturnType<typeof trackFetches>;

    beforeEach(() => {
      currentEnv = mockEnvironments({ app: mockApp() })[0];
      fetches = trackFetches();
    });

    afterEach(() => {
      fetches.restore();
    });

    // A slow response for the previous domain must not replace the current one.
    it("ignores the hours of a domain that is no longer selected", async () => {
      const first = mockDomain();
      const second = { ...mockDomain(), id: "2", domainName: "second.dev" };

      hourly(first, 9, 100);
      hourly(second, 3);

      wrapper = render(<Visitors {...props()} domain={first} ts="24h" />);
      wrapper.rerender(<Visitors {...props()} domain={second} ts="24h" />);

      await fetches.settle();

      expect(chart()).toEqual([
        { name: "2024-01-14 13:00", total: 3, unique: 1 },
      ]);
    });

    it("shows progress over the previous hours until the new ones arrive", async () => {
      const first = mockDomain();
      const second = { ...mockDomain(), id: "2", domainName: "second.dev" };

      hourly(first, 9);

      wrapper = render(<Visitors {...props()} domain={first} ts="24h" />);

      await fetches.settle();

      expect(chart()[0].total).toBe(9);
      expect(wrapper.queryByTestId("card-loading")).toBeNull();

      // Another domain is picked on a daily span, then 24 hours again.
      mockFetchAnalyticsReport({
        query: reportQuery(second, "7d"),
        response: mockReport(),
      });

      wrapper.rerender(<Visitors {...props()} domain={second} ts="7d" />);
      await fetches.settle();

      hourly(second, 3, 50);
      wrapper.rerender(<Visitors {...props()} domain={second} ts="24h" />);

      expect(wrapper.getByTestId("card-loading")).toBeTruthy();

      await fetches.settle();

      expect(wrapper.queryByTestId("card-loading")).toBeNull();
      expect(chart()[0].total).toBe(3);
    });

    it("clears the hours when they fail to load", async () => {
      const first = mockDomain();
      const second = { ...mockDomain(), id: "2", domainName: "second.dev" };

      hourly(first, 9);

      wrapper = render(<Visitors {...props()} domain={first} ts="24h" />);
      await fetches.settle();

      mockFetchVisitors({
        unique: "false",
        ts: "24h",
        envId: currentEnv.id,
        domainId: second.id,
        status: 500,
      });

      wrapper.rerender(<Visitors {...props()} domain={second} ts="24h" />);
      await fetches.settle();

      expect(
        wrapper.getByText(
          /Something went wrong while fetching unique visitors/,
        ),
      ).toBeTruthy();
      expect(chart()).toEqual([]);
    });
  });

  it("labels the span picker", async () => {
    createWrapper({});

    expect(wrapper.getByRole("group", { name: "Time span" })).toBeTruthy();

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });
  });

  it("labels hourly points in UTC and daily points by date", () => {
    expect(tooltipLabel("2024-01-14 13:00")).toBe("13:00 UTC");
    expect(tooltipLabel("2024-01-14")).toBe("2024-01-14");
  });

  it("formats durations", () => {
    expect(formatDuration(45)).toBe("45s");
    expect(formatDuration(95)).toBe("1m 35s");
    expect(formatDuration(600)).toBe("10m 00s");
  });
});
