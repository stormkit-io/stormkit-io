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
