import type { AnalyticsReport } from "./index.d";
import { render, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import mockDomain from "~/testing/data/mock_domain";
import { mockFetchAnalyticsReport } from "~/testing/nocks/nock_analytics";
import { reportRange } from "./actions";
import Countries from "./Countries";

// The map is an SVG library; the test checks the data it is given.
vi.mock("react-svg-worldmap", () => ({
  default: ({
    data,
    valueSuffix,
    size,
  }: {
    data: unknown;
    valueSuffix: string;
    size: number;
  }) => (
    <pre data-testid="world-map" data-suffix={valueSuffix} data-size={size}>
      {JSON.stringify(data)}
    </pre>
  ),
}));

const countriesReport = (): AnalyticsReport => ({
  metric: "countries",
  range: { from: "2024-01-08", to: "2024-01-14", days: 7 },
  totals: {},
  rows: [
    { country: "DE", visitors: 120 },
    { country: "TR", visitors: 80 },
  ],
});

describe("~/pages/apps/[id]/environments/[env-id]/analytics/Countries.tsx", () => {
  const env = mockEnvironments({ app: mockApp() })[0];
  const domain = mockDomain();

  it("maps visitors per country for the span", async () => {
    const scope = mockFetchAnalyticsReport({
      query: {
        envId: env.id!,
        domainId: domain.id,
        metric: "countries",
        limit: "50",
        ...reportRange("90d"),
      },
      response: countriesReport(),
    });

    const wrapper = render(
      <Countries environment={env} domain={domain} ts="90d" />,
    );

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(JSON.parse(wrapper.getByTestId("world-map").textContent!)).toEqual(
        [
          { country: "DE", value: 120 },
          { country: "TR", value: 80 },
        ],
      );
    });

    expect(wrapper.getByTestId("world-map").dataset.suffix).toBe("visitors");

    // Sized from the card, never wider than the cap.
    const size = Number(wrapper.getByTestId("world-map").dataset.size);
    expect(size).toBeGreaterThan(0);
    expect(size).toBeLessThanOrEqual(760);
    expect(
      wrapper.getByText("Visitors per country in the last 90 days."),
    ).toBeTruthy();
  });
});
