import type { AnalyticsReport, TimeSpan } from "./index.d";
import { render, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import mockDomain from "~/testing/data/mock_domain";
import { mockFetchAnalyticsReport } from "~/testing/nocks/nock_analytics";
import { reportRange } from "./actions";
import TopReferrers from "./TopReferrers";

const referrersReport = (): AnalyticsReport => ({
  metric: "referrers",
  range: { from: "2024-01-08", to: "2024-01-14", days: 7 },
  totals: { visits: 2400, direct: 300, internal: 1500 },
  rows: [
    { referrer: "google.com", channel: "search", visits: 400 },
    { referrer: "chatgpt.com", channel: "ai", visits: 150 },
    { referrer: "indiefounders.club", channel: "referral", visits: 50 },
  ],
});

describe("~/pages/apps/[id]/environments/[env-id]/analytics/TopReferrers.tsx", () => {
  const env = mockEnvironments({ app: mockApp() })[0];
  const domain = mockDomain();

  const mockReport = (ts: TimeSpan, response: AnalyticsReport) =>
    mockFetchAnalyticsReport({
      query: {
        envId: env.id!,
        domainId: domain.id,
        metric: "referrers",
        limit: "50",
        ...reportRange(ts),
      },
      response,
    });

  it("lists referring sites with their channel", async () => {
    const scope = mockReport("7d", referrersReport());
    const wrapper = render(
      <TopReferrers environment={env} domain={domain} ts="7d" />,
    );

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText("google.com")).toBeTruthy();
      expect(wrapper.getByText("Search")).toBeTruthy();
      expect(wrapper.getByText("chatgpt.com")).toBeTruthy();
      expect(wrapper.getByText("AI assistants")).toBeTruthy();
      expect(wrapper.getByText("Other sites")).toBeTruthy();
      expect(wrapper.getByText("400")).toBeTruthy();
    });
  });

  it("shows direct and in-site visits as totals instead of rows", async () => {
    mockReport("7d", referrersReport());

    const wrapper = render(
      <TopReferrers environment={env} domain={domain} ts="7d" />,
    );

    await waitFor(() => {
      expect(
        wrapper.getByText(
          /Top referring sites in the last 7 days\. 300 direct visits, 1,500 within this site\./,
        ),
      ).toBeTruthy();
    });
  });

  it("shows only the error when the report fails", async () => {
    const scope = mockFetchAnalyticsReport({
      query: {
        envId: env.id!,
        domainId: domain.id,
        metric: "referrers",
        limit: "50",
        ...reportRange("7d"),
      },
      status: 500,
      response: {},
    });

    const wrapper = render(
      <TopReferrers environment={env} domain={domain} ts="7d" />,
    );

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(
        wrapper.getByText("Something went wrong while fetching referrers."),
      ).toBeTruthy();
    });

    expect(wrapper.queryByText(/No visits from other sites/)).toBeNull();
  });

  it("clears the previous span's rows when the next one fails", async () => {
    mockReport("7d", referrersReport());

    mockFetchAnalyticsReport({
      query: {
        envId: env.id!,
        domainId: domain.id,
        metric: "referrers",
        limit: "50",
        ...reportRange("30d"),
      },
      status: 500,
      response: {},
    });

    const wrapper = render(
      <TopReferrers environment={env} domain={domain} ts="7d" />,
    );

    await waitFor(() => {
      expect(wrapper.getByText("google.com")).toBeTruthy();
    });

    wrapper.rerender(
      <TopReferrers environment={env} domain={domain} ts="30d" />,
    );

    await waitFor(() => {
      expect(
        wrapper.getByText("Something went wrong while fetching referrers."),
      ).toBeTruthy();
    });

    expect(wrapper.queryByText("google.com")).toBeNull();
  });

  it("says when no other site sent visits", async () => {
    const scope = mockReport("24h", { ...referrersReport(), rows: [] });
    const wrapper = render(
      <TopReferrers environment={env} domain={domain} ts="24h" />,
    );

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(
        wrapper.getByText(
          "No visits from other sites today (since midnight UTC).",
        ),
      ).toBeTruthy();
    });
  });
});
