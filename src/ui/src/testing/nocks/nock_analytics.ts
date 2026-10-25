import nock from "nock";

const endpoint = process.env.API_DOMAIN || "";

interface MockFetchVisitorsProps {
  ts?: string;
  unique?: "true" | "false";
  envId?: string;
  domainId: string;
  status?: number;
  response?: Record<string, { total: number; unique: number }>;
  // delay holds the response back, in milliseconds.
  delay?: number;
}

export const mockFetchVisitors = ({
  delay = 0,
  unique,
  ts,
  envId,
  status = 200,
  domainId,
  response,
}: MockFetchVisitorsProps) => {
  return nock(endpoint)
    .get(
      `/analytics/visitors?unique=${unique}&envId=${envId}&ts=${ts}&domainId=${domainId}`,
    )
    .delay(delay)
    .reply(status, response);
};

interface MockFetchAnalyticsReportProps {
  query: Record<string, string>;
  status?: number;
  response?: unknown;
  // delay holds the response back, in milliseconds.
  delay?: number;
}

// mockFetchAnalyticsReport mocks GET /v1/analytics with exactly these query
// parameters.
export const mockFetchAnalyticsReport = ({
  query,
  status = 200,
  response,
  delay = 0,
}: MockFetchAnalyticsReportProps) => {
  return nock(endpoint)
    .get("/v1/analytics")
    .query(query)
    .delay(delay)
    .reply(status, response);
};

interface MockFetchEventsProps {
  ts?: string;
  envId?: string;
  domainId: string;
  status?: number;
  response?: { name: string; total: number; unique: number }[];
  // delay holds the response back, in milliseconds.
  delay?: number;
}

export const mockFetchEvents = ({
  delay = 0,
  ts,
  envId,
  status = 200,
  domainId,
  response,
}: MockFetchEventsProps) => {
  return nock(endpoint)
    .get(`/analytics/events?envId=${envId}&domainId=${domainId}&ts=${ts}`)
    .delay(delay)
    .reply(status, response);
};

interface MockFetchEventPropertiesProps {
  ts?: string;
  envId?: string;
  domainId: string;
  event: string;
  status?: number;
  response?: string[];
  // delay holds the response back, in milliseconds.
  delay?: number;
}

export const mockFetchEventProperties = ({
  delay = 0,
  ts,
  envId,
  status = 200,
  domainId,
  event,
  response,
}: MockFetchEventPropertiesProps) => {
  return nock(endpoint)
    .get(
      `/analytics/events/properties?envId=${envId}&domainId=${domainId}&event=${event}&ts=${ts}`,
    )
    .delay(delay)
    .reply(status, response);
};

interface MockFetchEventBreakdownProps {
  ts?: string;
  envId?: string;
  domainId: string;
  event: string;
  property: string;
  status?: number;
  response?: { name: string; total: number; unique: number }[];
  // delay holds the response back, in milliseconds.
  delay?: number;
}

export const mockFetchEventBreakdown = ({
  delay = 0,
  ts,
  envId,
  status = 200,
  domainId,
  event,
  property,
  response,
}: MockFetchEventBreakdownProps) => {
  return nock(endpoint)
    .get(
      `/analytics/events/breakdown?envId=${envId}&domainId=${domainId}&event=${event}&property=${property}&ts=${ts}`,
    )
    .delay(delay)
    .reply(status, response);
};

interface MockAnalyticsStatusProps {
  appId: string;
  envId: string;
  status?: number;
  enabled: boolean;
}

export const mockFetchAnalyticsStatus = ({
  appId,
  envId,
  status = 200,
  enabled,
}: MockAnalyticsStatusProps) => {
  return nock(endpoint)
    .get(`/snippets/analytics?appId=${appId}&envId=${envId}`)
    .reply(status, { enabled });
};

interface MockToggleAnalyticsProps {
  appId: string;
  envId: string;
  status?: number;
}

export const mockEnableAnalytics = ({
  appId,
  envId,
  status = 200,
}: MockToggleAnalyticsProps) => {
  return nock(endpoint)
    .post(`/snippets/analytics`, { appId, envId })
    .reply(status, { snippet: {} });
};

export const mockDisableAnalytics = ({
  appId,
  envId,
  status = 200,
}: MockToggleAnalyticsProps) => {
  return nock(endpoint)
    .delete(`/snippets/analytics?appId=${appId}&envId=${envId}`)
    .reply(status, {});
};
