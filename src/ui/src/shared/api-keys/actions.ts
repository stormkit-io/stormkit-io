import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

interface FetchAPIKeyProps {
  appId?: string;
  envId?: string;
  teamId?: string;
  userId?: string;
  refreshToken?: number;
}

export const useFetchAPIKeys = ({
  appId = "",
  envId = "",
  teamId = "",
  userId = "",
  refreshToken,
}: FetchAPIKeyProps) => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const [keys, setKeys] = useState<APIKey[]>([]);

  useEffect(() => {
    setLoading(true);
    setError(undefined);

    api
      .fetch<{ keys: APIKey[] }>(
        `/api-keys?appId=${appId}&envId=${envId}&teamId=${teamId}&userId=${userId}`,
      )
      .then(({ keys }) => {
        setKeys(keys);
      })
      .catch(e => {
        setError("Something went wrong while fetching api key");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [appId, envId, teamId, userId, refreshToken]);

  return { loading, error, keys, setKeys };
};

export const deleteAPIKey = (apiKey: APIKey) => {
  return api.delete<{ keys: APIKey[] }>(`/api-keys?keyId=${apiKey.id}`);
};

interface GenerateNewAPIKeyProps {
  appId?: string;
  envId?: string;
  teamId?: string;
  userId?: string;
  name: string;
  scope: string;
}

export const generateNewAPIKey = ({
  name,
  scope,
  appId,
  envId,
  teamId,
  userId,
}: GenerateNewAPIKeyProps) => {
  return api.post<APIKey>("/api-keys", {
    appId,
    envId,
    teamId,
    userId,
    name,
    scope,
  });
};
