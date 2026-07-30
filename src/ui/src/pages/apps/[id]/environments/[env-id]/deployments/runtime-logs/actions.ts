import { useState, useEffect } from "react";
import api from "~/utils/api/Api";

interface FetchDeploymentRuntimeLogsProps {
  envId: string;
  keySetId?: string;
  deploymentId: string;
  sort?: "asc" | "desc";
  reset?: boolean;
}

export interface Log {
  id: string;
  appId: string;
  deploymentId: string;
  timestamp: string;
  data: string;
}

export const useFetchDeploymentRuntimeLogs = ({
  envId,
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
      envId,
      sort,
    });

    if (sort === "asc" && keySetId) {
      params.set("beforeId", keySetId);
    } else if (sort === "desc" && keySetId) {
      params.set("afterId", keySetId);
    }

    api
      .fetch<{ logs: Log[]; hasNextPage: boolean }>(
        `/v1/deployments/${deploymentId}/runtime-logs?${params.toString()}`
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
  }, [envId, deploymentId, keySetId, sort]);

  return { logs, error, loading, hasNextPage };
};
