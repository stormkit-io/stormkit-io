import nock from "nock";

const endpoint = process.env.API_DOMAIN || "";

interface MockFetchFeatureFlagsProps {
  appId: string;
  envId: string;
  status?: number;
  response: FeatureFlag[];
}

export const mockFetchFeatureFlags = ({
  appId,
  envId,
  status = 200,
  response = [],
}: MockFetchFeatureFlagsProps) => {
  return nock(endpoint)
    .get(`/apps/${appId}/envs/${envId}/flags`)
    .reply(status, response);
};

interface MockUpsertFeatureFlagsProps {
  appId: string;
  envId: string;
  flagName: string;
  flagValue: boolean;
  status?: number;
  response?: { ok: boolean };
}

export const mockUpsertFeatureFlags = ({
  appId,
  envId,
  flagName,
  flagValue,
  status = 200,
  response = { ok: true },
}: MockUpsertFeatureFlagsProps) => {
  return nock(endpoint)
    .post(`/apps/flags`, {
      appId,
      envId,
      flagName,
      flagValue,
    })
    .reply(status, response);
};

interface MockDeleteFeatureFlagsProps {
  appId: string;
  envId: string;
  flagName: string;
  status?: number;
  response?: { ok: boolean };
}

export const mockDeleteFeatureFlags = ({
  appId,
  envId,
  flagName,
  status = 200,
  response = { ok: true },
}: MockDeleteFeatureFlagsProps) => {
  return nock(endpoint)
    .delete(`/apps/flags`, {
      appId,
      envId,
      flagName,
    })
    .reply(status, response);
};
