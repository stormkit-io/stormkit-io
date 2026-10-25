import type { AnalyticsReport, TimeSpan } from "./index.d";
import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

interface FetchLatestParams<T> {
  url: string;
  onData: (data: T) => void;
  onError: () => void;
  onSettled: () => void;
}

// fetchLatest requests url and calls back unless the returned cleanup ran
// first, so a slow response of a previous request never replaces the current
// one's. Effects return the cleanup.
const fetchLatest = <T>({
  url,
  onData,
  onError,
  onSettled,
}: FetchLatestParams<T>) => {
  let current = true;

  api
    .fetch<T>(url)
    .then(data => {
      if (current) {
        onData(data);
      }
    })
    .catch(() => {
      if (current) {
        onError();
      }
    })
    .finally(() => {
      if (current) {
        onSettled();
      }
    });

  return () => {
    current = false;
  };
};

interface FetchHourlyVisitorsProps {
  envId: string;
  domainId?: string;
  refreshToken?: number;
}

interface VisitorsChartData {
  name: string;
  total: number;
  unique: number;
}

// useFetchVisitors reads the hourly page views and visitors of the last 24
// hours. Longer spans use useFetchVisitorsReport.
export const useFetchVisitors = ({
  envId,
  refreshToken,
  domainId,
}: FetchHourlyVisitorsProps) => {
  const [visitors, setVisitors] = useState<VisitorsChartData[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setError("");

    if (!domainId) {
      return;
    }

    // The previous hours stay on screen under a progress bar until these
    // arrive.
    setLoading(true);

    return fetchLatest<Record<string, { total: number; unique: number }>>({
      url: `/analytics/visitors?unique=false&envId=${envId}&ts=24h&domainId=${domainId}`,
      onData: data => {
        setVisitors(
          Object.keys(data).map(name => ({
            name,
            total: data[name]?.total || 0,
            unique: data[name]?.unique || 0,
          })),
        );
      },
      onError: () => {
        setVisitors([]);
        setError("Something went wrong while fetching unique visitors.");
      },
      onSettled: () => setLoading(false),
    });
  }, [refreshToken, envId, domainId]);

  return { visitors, error, loading };
};

// utcDate returns the UTC date `offset` days from today as YYYY-MM-DD.
const utcDate = (offset: number): string => {
  const date = new Date();
  date.setUTCDate(date.getUTCDate() + offset);

  return date.toISOString().slice(0, 10);
};

// reportRange returns the report range of a time span. The last 24 hours read
// today so far; longer spans read complete days ending yesterday (UTC), so a
// partly elapsed today is never compared with full previous days.
export const reportRange = (ts: TimeSpan) =>
  ts === "24h" ? { span: "1d", to: utcDate(0) } : { span: ts, to: utcDate(-1) };

export interface AnalyticsReportQuery {
  metric: string;
  ts: TimeSpan;
  path?: string;
  pathPrefix?: string;
  limit?: number;
  compare?: boolean;
}

interface FetchAnalyticsReportProps {
  envId: string;
  domainId?: string;
  query: AnalyticsReportQuery;
  skip?: boolean;
}

// useFetchAnalyticsReport reads a GET /v1/analytics report.
export const useFetchAnalyticsReport = ({
  envId,
  domainId,
  query,
  skip,
}: FetchAnalyticsReportProps) => {
  const [report, setReport] = useState<AnalyticsReport>();
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const key = JSON.stringify(query);

  useEffect(() => {
    setError("");

    if (!domainId || skip) {
      return;
    }

    const { span, to } = reportRange(query.ts);
    const params = new URLSearchParams({
      envId,
      domainId,
      metric: query.metric,
      span,
      to,
    });

    if (query.path) {
      params.set("path", query.path);
    }

    if (query.pathPrefix) {
      params.set("pathPrefix", query.pathPrefix);
    }

    if (query.limit) {
      params.set("limit", `${query.limit}`);
    }

    if (query.compare) {
      params.set("compare", "previous");
    }

    setLoading(true);

    return fetchLatest<AnalyticsReport>({
      url: `/v1/analytics?${params.toString()}`,
      onData: setReport,
      onError: () => {
        // Keeping the previous report would show its numbers under the new
        // query's labels.
        setReport(undefined);
        setError(`Something went wrong while fetching ${query.metric}.`);
      },
      onSettled: () => setLoading(false),
    });
  }, [envId, domainId, key, skip]);

  return { report, error, loading };
};

interface FetchVisitorsReportProps {
  envId: string;
  domainId?: string;
  ts: Exclude<TimeSpan, "24h">;
}

// useFetchVisitorsReport reads the daily visitors report for a span, with the
// previous period of the same length for comparison.
export const useFetchVisitorsReport = ({
  envId,
  domainId,
  ts,
}: FetchVisitorsReportProps) =>
  useFetchAnalyticsReport({
    envId,
    domainId,
    query: { metric: "visitors", ts, compare: true },
  });

interface FetchEventsProps {
  envId: string;
  domainId?: string;
  ts?: TimeSpan;
}

interface AnalyticsEvent {
  name: string;
  total: number;
  unique: number;
}

export const useFetchEvents = ({
  envId,
  domainId,
  ts = "30d",
}: FetchEventsProps) => {
  const [events, setEvents] = useState<AnalyticsEvent[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!domainId) {
      return;
    }

    setError("");
    setLoading(true);

    return fetchLatest<AnalyticsEvent[]>({
      url: `/analytics/events?envId=${envId}&domainId=${domainId}&ts=${ts}`,
      onData: data => setEvents(Array.isArray(data) ? data : []),
      onError: () => {
        setEvents([]);
        setError("Something went wrong while fetching events.");
      },
      onSettled: () => setLoading(false),
    });
  }, [envId, domainId, ts]);

  return { events, loading, error };
};

interface FetchEventPropertiesProps {
  envId: string;
  domainId?: string;
  event: string;
  ts?: TimeSpan;
  skip?: boolean;
}

export const useFetchEventProperties = ({
  envId,
  domainId,
  event,
  ts = "30d",
  skip,
}: FetchEventPropertiesProps) => {
  const [properties, setProperties] = useState<string[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (skip || !domainId || !event) {
      return;
    }

    setError("");
    setLoading(true);

    return fetchLatest<string[]>({
      url: `/analytics/events/properties?envId=${envId}&domainId=${domainId}&event=${encodeURIComponent(
        event,
      )}&ts=${ts}`,
      onData: data => setProperties(Array.isArray(data) ? data : []),
      onError: () => {
        setProperties([]);
        setError("Something went wrong while fetching event properties.");
      },
      onSettled: () => setLoading(false),
    });
  }, [envId, domainId, event, ts, skip]);

  return { properties, loading, error };
};

interface FetchEventBreakdownProps {
  envId: string;
  domainId?: string;
  event: string;
  property: string;
  ts?: TimeSpan;
  skip?: boolean;
}

export const useFetchEventBreakdown = ({
  envId,
  domainId,
  event,
  property,
  ts = "30d",
  skip,
}: FetchEventBreakdownProps) => {
  const [breakdown, setBreakdown] = useState<AnalyticsEvent[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (skip || !domainId || !event || !property) {
      return;
    }

    setError("");
    setLoading(true);

    return fetchLatest<AnalyticsEvent[]>({
      url: `/analytics/events/breakdown?envId=${envId}&domainId=${domainId}&event=${encodeURIComponent(
        event,
      )}&property=${encodeURIComponent(property)}&ts=${ts}`,
      onData: data => setBreakdown(Array.isArray(data) ? data : []),
      onError: () => {
        setBreakdown([]);
        setError("Something went wrong while fetching the event breakdown.");
      },
      onSettled: () => setLoading(false),
    });
  }, [envId, domainId, event, property, ts, skip]);

  return { breakdown, loading, error };
};

interface AnalyticsStatusProps {
  appId: string;
  envId: string;
  refreshToken?: number;
}

export const useFetchAnalyticsStatus = ({
  appId,
  envId,
  refreshToken,
}: AnalyticsStatusProps) => {
  const [enabled, setEnabled] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);

    api
      .fetch<{ enabled: boolean }>(
        `/snippets/analytics?appId=${appId}&envId=${envId}`,
      )
      .then(data => {
        setEnabled(Boolean(data.enabled));
      })
      .catch(() => {
        setEnabled(false);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [appId, envId, refreshToken]);

  return { enabled, loading };
};

interface ToggleAnalyticsProps {
  appId: string;
  envId: string;
}

export const enableAnalytics = ({
  appId,
  envId,
}: ToggleAnalyticsProps): Promise<void> => {
  return api.post(`/snippets/analytics`, { appId, envId });
};

export const disableAnalytics = ({
  appId,
  envId,
}: ToggleAnalyticsProps): Promise<void> => {
  return api.delete(`/snippets/analytics?appId=${appId}&envId=${envId}`);
};
