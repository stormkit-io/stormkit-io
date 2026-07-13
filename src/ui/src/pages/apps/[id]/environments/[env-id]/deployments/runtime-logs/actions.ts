import { useState, useEffect } from "react";
import api from "~/utils/api/Api";

interface FetchDeploymentRuntimeLogsProps {
  appId: string;
  keySetId?: string;
  deploymentId: string;
  sort?: "asc" | "desc";
  reset?: boolean;
}

export interface Log {
  id: string;
  appId: string;
  envId: string;
  deploymentId: string;
  timestamp: string;
  data: string;
}

export const useFetchDeploymentRuntimeLogs = ({
  appId,
  deploymentId,
  keySetId,
  sort = "asc",
  reset,
}: FetchDeploymentRuntimeLogsProps) => {
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(true);
  const [logs, setLogs] = useState<Log[]>([]);
  const [hasNextPage, setHasNextPage] = useState(false);

  useEffect(() => {
    setLoading(true);
    setError(undefined);

    const params = new URLSearchParams({
      sort,
      deploymentId,
    });

    if (sort === "asc" && keySetId) {
      params.set("beforeId", keySetId);
    } else if (sort === "desc" && keySetId) {
      params.set("afterId", keySetId);
    }

    api
      .fetch<{ logs: Log[]; hasNextPage: boolean }>(
        `/app/${appId}/logs?${params.toString()}`
      )
      .then(data => {
        if (reset) {
          setLogs(data.logs);
        } else {
          setLogs([...logs, ...data.logs]);
        }

        setHasNextPage(data.hasNextPage);
      })
      .catch(() => {
        setError("Something went wrong while fetching logs.");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [appId, deploymentId, keySetId, sort]);

  return { logs, error, loading, hasNextPage };
};
