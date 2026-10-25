import type { AnalyticsReport, TimeSpan } from "./index.d";
import { fireEvent, render, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import mockDomain from "~/testing/data/mock_domain";
import { mockFetchAnalyticsReport } from "~/testing/nocks/nock_analytics";
import { reportRange } from "./actions";
import TopPaths from "./TopPaths";

const pathsReport = (): AnalyticsReport => ({
  metric: "paths",
  range: { from: "2024-01-08", to: "2024-01-14", days: 7 },
  totals: { pageViews: 1300 },
  rows: [
    { path: "/blog/a", pageViews: 900, visitors: 300, avgTimeOnPage: 95 },
    { path: "/pricing", pageViews: 400, visitors: 120 },
    { path: "(other)", pageViews: 30, visitors: 30 },
  ],
});

const channelsReport = (): AnalyticsReport => ({
  metric: "channels",
  range: { from: "2024-01-08", to: "2024-01-14", days: 7 },
  totals: { pageViews: 900, visitors: 300, visits: 900, avgTimeOnPage: 95 },
  rows: [
    { channel: "search", visits: 500 },
    { channel: "ai", visits: 300 },
    { channel: "internal", visits: 100 },
  ],
});

describe("~/pages/apps/[id]/environments/[env-id]/analytics/TopPaths.tsx", () => {
  const env = mockEnvironments({ app: mockApp() })[0];
  const domain = mockDomain();

  const query = (ts: TimeSpan, extra: Record<string, string> = {}) => ({
    envId: env.id!,
    domainId: domain.id,
    ...reportRange(ts),
    ...extra,
  });

  const renderPaths = (ts: TimeSpan = "7d") =>
    render(<TopPaths environment={env} domain={domain} ts={ts} />);

  it("lists paths with their views, visitors and time on page", async () => {
    const scope = mockFetchAnalyticsReport({
      query: query("7d", { metric: "paths", limit: "50" }),
      response: pathsReport(),
    });

    const wrapper = renderPaths();

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText("/blog/a")).toBeTruthy();
      expect(
        wrapper.getByText("900 views · 300 visitors · 1m 35s"),
      ).toBeTruthy();
      expect(wrapper.getByText("400 views · 120 visitors")).toBeTruthy();
      expect(
        wrapper.getByText("Top visited paths in the last 7 days."),
      ).toBeTruthy();
    });
  });

  it("reads today for the 24 hours span", async () => {
    const scope = mockFetchAnalyticsReport({
      query: query("24h", { metric: "paths", limit: "50" }),
      response: pathsReport(),
    });

    const wrapper = renderPaths("24h");

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(
        wrapper.getByText("Top visited paths today (since midnight UTC)."),
      ).toBeTruthy();
    });
  });

  it("filters by a path prefix, adding the leading slash", async () => {
    mockFetchAnalyticsReport({
      query: query("7d", { metric: "paths", limit: "50" }),
      response: pathsReport(),
    });

    const filtered = mockFetchAnalyticsReport({
      query: query("7d", {
        metric: "paths",
        limit: "50",
        pathPrefix: "/blog/",
      }),
      response: { ...pathsReport(), rows: [pathsReport().rows![0]] },
    });

    const wrapper = renderPaths();
    const input = wrapper.getByLabelText("Path prefix");

    fireEvent.change(input, { target: { value: "blog/" } });
    fireEvent.keyDown(input, { key: "Enter" });

    await waitFor(() => {
      expect(filtered.isDone()).toBe(true);
      expect(wrapper.queryByText("/pricing")).toBeNull();
    });
  });

  it("shows where a path's visitors came from", async () => {
    mockFetchAnalyticsReport({
      query: query("7d", { metric: "paths", limit: "50" }),
      response: pathsReport(),
    });

    const channels = mockFetchAnalyticsReport({
      query: query("7d", { metric: "channels", path: "/blog/a" }),
      response: channelsReport(),
    });

    const wrapper = renderPaths();

    await waitFor(() => {
      expect(wrapper.getByText("/blog/a")).toBeTruthy();
    });

    fireEvent.click(
      wrapper.getByLabelText("Show where /blog/a visitors came from"),
    );

    await waitFor(() => {
      expect(channels.isDone()).toBe(true);
      expect(wrapper.getByText("Search")).toBeTruthy();
      expect(wrapper.getByText("AI assistants")).toBeTruthy();
      expect(wrapper.getByText("Within this site")).toBeTruthy();
      expect(wrapper.getByText("500")).toBeTruthy();
    });

    fireEvent.click(wrapper.getByLabelText("Back to paths"));

    await waitFor(() => {
      expect(wrapper.getByText("/pricing")).toBeTruthy();
    });
  });

  it("does not offer a drill-down into (other)", async () => {
    mockFetchAnalyticsReport({
      query: query("7d", { metric: "paths", limit: "50" }),
      response: pathsReport(),
    });

    const wrapper = renderPaths();

    await waitFor(() => {
      expect(wrapper.getByText("(other)")).toBeTruthy();
    });

    expect(
      wrapper.queryByLabelText("Show where (other) visitors came from"),
    ).toBeNull();
    expect(
      wrapper.getByLabelText("Show where /pricing visitors came from"),
    ).toBeTruthy();
  });

  it("shows progress while a new span loads over the previous rows", async () => {
    mockFetchAnalyticsReport({
      query: query("7d", { metric: "paths", limit: "50" }),
      response: pathsReport(),
    });

    mockFetchAnalyticsReport({
      query: query("90d", { metric: "paths", limit: "50" }),
      response: pathsReport(),
      delay: 200,
    });

    const wrapper = renderPaths();

    await waitFor(() => {
      expect(wrapper.getByText("/blog/a")).toBeTruthy();
    });

    wrapper.rerender(<TopPaths environment={env} domain={domain} ts="90d" />);

    await waitFor(() => {
      expect(wrapper.getByRole("progressbar")).toBeTruthy();
    });

    await waitFor(() => {
      expect(wrapper.queryByRole("progressbar")).toBeNull();
    });
  });

  it("says when there are no page views", async () => {
    const scope = mockFetchAnalyticsReport({
      query: query("30d", { metric: "paths", limit: "50" }),
      response: { ...pathsReport(), rows: [] },
    });

    const wrapper = renderPaths("30d");

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(
        wrapper.getByText(/No page views in the last 30 days/),
      ).toBeTruthy();
    });
  });
});
