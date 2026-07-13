import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

interface Filters {
  teamId?: string;
  branch?: string;
  published?: boolean;
  status?: boolean;
}

interface UseFetchDeploymentsProps {
  from?: number;
  refreshToken?: number;
  filters?: Filters;
}

export const useFetchDeployments = ({
  from,
  filters,
  refreshToken,
}: UseFetchDeploymentsProps) => {
  const [deployments, setDeployments] = useState<DeploymentV2[]>([]);
  const [hasNextPage, setHasNextPage] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let unmounted = false;

    setLoading(true);
    setError(null);

    const params = new URLSearchParams(JSON.parse(JSON.stringify(filters)));

    api
      .fetch<{ deployments: DeploymentV2[]; hasNextPage: boolean }>(
        "/my/deployments?" + params.toString()
      )
      .then(res => {
        if (unmounted !== true) {
          setHasNextPage(res.hasNextPage);
          setDeployments(
            from && from > 0
              ? [...deployments, ...res.deployments]
              : res.deployments
          );
        }
      })
      .finally(() => {
        if (unmounted !== true) {
          setLoading(false);
        }
      });

    return () => {
      unmounted = true;
    };
  }, [refreshToken]);

  return {
    deployments,
    error,
    loading,
    hasNextPage,
  };
};
