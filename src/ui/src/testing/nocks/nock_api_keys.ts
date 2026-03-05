import nock from "nock";
import * as data from "../data";

const endpoint = process.env.API_DOMAIN || "";

interface MockFetchAPIKeysProps {
  appId?: string;
  envId?: string;
  teamId?: string;
  userId?: string;
  status?: number;
  response?: { keys: APIKey[] };
}

export const mockFetchAPIKeys = ({
  appId = "",
  envId = "",
  teamId = "",
  userId = "",
  status = 200,
  response = data.mockAPIKeysResponse(),
}: MockFetchAPIKeysProps) => {
  return nock(endpoint)
    .get(
      `/api-keys?appId=${appId}&envId=${envId}&teamId=${teamId}&userId=${userId}`,
    )
    .reply(status, response);
};

interface MockGenerateAPIKeyProps {
  name: string;
  scope: string;
  appId?: string;
  envId?: string;
  teamId?: string;
  userId?: string;
  status?: number;
  response?: APIKey;
}

export const mockGenerateAPIKey = ({
  name,
  scope,
  appId,
  envId,
  teamId,
  userId,
  status = 201,
  response = {
    id: "1234567890",
    name,
    scope: scope as APIKey["scope"],
    appId: appId || "",
    envId: envId || "",
    token:
      "SK_newtoken1234567890abcdefghijklmnopqrstuvwxyz1234567890abcdefghij",
  },
}: MockGenerateAPIKeyProps) => {
  // JSON.stringify drops undefined values, so only include defined fields in
  // the body matcher to avoid nock mismatches.
  const body: Record<string, string> = { name, scope };
  if (appId !== undefined) body.appId = appId;
  if (envId !== undefined) body.envId = envId;
  if (teamId !== undefined) body.teamId = teamId;
  if (userId !== undefined) body.userId = userId;

  return nock(endpoint).post("/api-keys", body).reply(status, response);
};

interface MockDeleteAPIKeyProps {
  keyId: string;
  status?: number;
}

export const mockDeleteAPIKey = ({
  keyId,
  status = 200,
}: MockDeleteAPIKeyProps) => {
  return nock(endpoint)
    .delete(`/api-keys?keyId=${keyId}`)
    .reply(status, { ok: true });
};
