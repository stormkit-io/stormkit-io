import type { AnalyticsReport } from "~/pages/apps/[id]/environments/[env-id]/analytics/index.d";

export default (): AnalyticsReport => ({
  metric: "visitors",
  range: { from: "2024-01-08", to: "2024-01-14", days: 7 },
  totals: { pageViews: 1200, visitors: 400, avgTimeOnPage: 95 },
  series: [
    { date: "2024-01-12", pageViews: 300, visitors: 100, avgTimeOnPage: 80 },
    { date: "2024-01-13", pageViews: 500, visitors: 150 },
    { date: "2024-01-14", pageViews: 400, visitors: 150, avgTimeOnPage: 110 },
  ],
  previous: {
    range: { from: "2024-01-01", to: "2024-01-07", days: 7 },
    totals: { pageViews: 1000, visitors: 500, avgTimeOnPage: 95 },
  },
  notes: [],
});
