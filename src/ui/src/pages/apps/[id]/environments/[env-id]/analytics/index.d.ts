export type TimeSpan = "24h" | "7d" | "30d" | "90d" | "365d";

// AnalyticsTotals mirrors the totals of a GET /v1/analytics report. Only the
// counts relevant to the metric are present.
export interface AnalyticsTotals {
  pageViews?: number;
  visitors?: number;
  pageVisitors?: number;
  visits?: number;
  direct?: number;
  internal?: number;
  count?: number;
  avgTimeOnPage?: number;
}

export interface AnalyticsDay {
  date: string;
  pageViews: number;
  visitors?: number;
  pageVisitors?: number;
  avgTimeOnPage?: number;
}

export interface AnalyticsReport {
  metric: string;
  range: { from: string; to: string; days: number };
  totals: AnalyticsTotals;
  series?: AnalyticsDay[];
  previous?: {
    range: { from: string; to: string; days: number };
    totals: AnalyticsTotals;
  };
  notes?: string[];
}
