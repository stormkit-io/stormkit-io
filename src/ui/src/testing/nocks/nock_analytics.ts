import nock from "nock";

const endpoint = process.env.API_DOMAIN || "";

interface MockFetchVisitorsProps {
  ts?: "24h" | "7d" | "30d";
  unique?: "true" | "false";
  envId?: string;
  domainId: string;
  status?: number;
  response?: Record<string, { total: number; unique: number }>;
}

export const mockFetchVisitors = ({
  unique,
  ts,
  envId,
  status = 200,
  domainId,
  response,
}: MockFetchVisitorsProps) => {
  return nock(endpoint)
    .get(
      `/analytics/visitors?unique=${unique}&envId=${envId}&ts=${ts}&domainId=${domainId}`
    )
    .reply(status, response);
};

interface MockFetchEventsProps {
  ts?: "24h" | "7d" | "30d";
  envId?: string;
  domainId: string;
  status?: number;
  response?: { name: string; total: number; unique: number }[];
}

export const mockFetchEvents = ({
  ts,
  envId,
  status = 200,
  domainId,
  response,
}: MockFetchEventsProps) => {
  return nock(endpoint)
    .get(`/analytics/events?envId=${envId}&domainId=${domainId}&ts=${ts}`)
    .reply(status, response);
};

interface MockFetchEventPropertiesProps {
  ts?: "24h" | "7d" | "30d";
  envId?: string;
  domainId: string;
  event: string;
  status?: number;
  response?: string[];
}

export const mockFetchEventProperties = ({
  ts,
  envId,
  status = 200,
  domainId,
  event,
  response,
}: MockFetchEventPropertiesProps) => {
  return nock(endpoint)
    .get(
      `/analytics/events/properties?envId=${envId}&domainId=${domainId}&event=${event}&ts=${ts}`
    )
    .reply(status, response);
};

interface MockFetchEventBreakdownProps {
  ts?: "24h" | "7d" | "30d";
  envId?: string;
  domainId: string;
  event: string;
  property: string;
  status?: number;
  response?: { name: string; total: number; unique: number }[];
}

export const mockFetchEventBreakdown = ({
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
      `/analytics/events/breakdown?envId=${envId}&domainId=${domainId}&event=${event}&property=${property}&ts=${ts}`
    )
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
