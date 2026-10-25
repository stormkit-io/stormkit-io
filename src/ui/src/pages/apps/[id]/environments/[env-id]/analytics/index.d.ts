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

export interface AnalyticsRow {
  path?: string;
  referrer?: string;
  channel?: string;
  country?: string;
  utmSource?: string;
  utmMedium?: string;
  utmCampaign?: string;
  ref?: string;
  event?: string;
  pageViews?: number;
  visits?: number;
  visitors?: number;
  pageVisitors?: number;
  count?: number;
  avgTimeOnPage?: number;
  channels?: Record<string, number>;
  previous?: number;
}

export interface AnalyticsReport {
  metric: string;
  range: { from: string; to: string; days: number };
  totals: AnalyticsTotals;
  series?: AnalyticsDay[];
  rows?: AnalyticsRow[];
  previous?: {
    range: { from: string; to: string; days: number };
    totals: AnalyticsTotals;
  };
  notes?: string[];
}
