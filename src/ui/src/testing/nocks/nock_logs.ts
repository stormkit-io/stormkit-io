import type { Log } from "~/pages/apps/[id]/environments/[env-id]/deployments/runtime-logs/actions";
import nock from "nock";

const endpoint = process.env.API_DOMAIN || "";

interface FetchDeploymentLogsProps {
  appId: string;
  deploymentId: string;
  status?: number;
  keySetId?: string;
  sort?: "asc" | "desc";
  response: {
    logs: Log[];
    hasNextPage: boolean;
  };
}

export const mockFetchDeploymentLogs = ({
  appId,
  deploymentId,
  status = 200,
  keySetId,
  sort = "desc",
  response = {
    logs: [],
    hasNextPage: false,
  },
}: FetchDeploymentLogsProps) => {
  const params = new URLSearchParams({
    sort,
    deploymentId,
  });

  if (sort === "asc" && keySetId) {
    params.set("beforeId", keySetId);
  } else if (sort === "desc" && keySetId) {
    params.set("afterId", keySetId);
  }

  return nock(endpoint)
    .get(`/app/${appId}/logs?${params.toString()}`)
    .reply(status, response);
};
