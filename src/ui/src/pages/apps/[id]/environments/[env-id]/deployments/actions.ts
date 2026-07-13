import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

interface DeleteForeverProps {
  appId: string;
  deploymentId: string;
}

export const deleteForever = ({
  appId,
  deploymentId,
}: DeleteForeverProps): Promise<void> => {
  return api.delete(`/app/deploy`, { deploymentId, appId });
};

interface PublishDeploymentsProps {
  app: App;
  envId: string;
  percentages: Record<string, number>;
}

interface PublishInfo {
  deploymentId: string;
  percentage: number;
}

export const publishDeployments = ({
  app,
  percentages,
  envId,
}: PublishDeploymentsProps): Promise<void> => {
  const publish: Array<PublishInfo> = [];
  let total = 0;

  Object.keys(percentages).forEach(deploymentId => {
    const percentage = percentages[deploymentId];
    total = total + percentage;

    publish.push({
      percentage,
      deploymentId,
    });
  });

  if (total !== 100) {
    return Promise.reject(
      `The sum of percentages has to be 100. Currently it is ${total}.`,
    );
  }

  return api.post(`/app/deployments/publish`, {
    appId: app.id,
    envId,
    publish,
  });
};

interface FetchManifestProps {
  deploymentId: string;
  appId: string;
}

interface FetchManifestReturnValue {
  manifest: Manifest;
  loading: boolean;
  error: string | null;
}

export const useFetchManifest = ({
  appId,
  deploymentId,
}: FetchManifestProps): FetchManifestReturnValue => {
  const [manifest, setManifest] = useState<Manifest>({});
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    setError(null);

    api
      .fetch<{ manifest: Manifest }>(`/app/${appId}/manifest/${deploymentId}`, {
        method: "GET",
      })
      .then(data => setManifest(data.manifest))
      .catch(e => setError(e))
      .finally(() => setLoading(false));
  }, []);

  return {
    loading,
    error,
    manifest,
  };
};

interface Filters {
  envId?: string;
  branch?: string;
  published?: boolean;
  status?: boolean;
}

interface UseFetchDeploymentsProps {
  app: App;
  refreshToken: number;
  from?: number;
  filters?: Filters;
}

export const useFetchDeployments = ({
  app,
  from,
  refreshToken,
  filters,
}: UseFetchDeploymentsProps) => {
  const [deployments, setDeployments] = useState<DeploymentV2[]>([]);
  // const [hasNextPage, setHasNextPage] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const { envId, status, published, branch } = filters || {};

  useEffect(() => {
    let unmounted = false;

    if (envId === "") {
      setDeployments([]);
      return;
    }

    setLoading(true);
    setError(null);

    const params = new URLSearchParams(JSON.parse(JSON.stringify(filters)));

    api
      .fetch<{ deployments: DeploymentV2[] }>(
        `/my/deployments?${params.toString()}`,
      )
      .then(res => {
        if (unmounted !== true) {
          setDeployments(
            from && from > 0
              ? [...deployments, ...res.deployments]
              : res.deployments,
          );
        }
      })
      .catch(() => {
        setError("Something went wrong while fetching deployments.");
      })
      .finally(() => {
        if (unmounted !== true) {
          setLoading(false);
        }
      });

    return () => {
      unmounted = true;
    };
  }, [app.id, refreshToken, envId, published, branch, status, from]);

  return {
    deployments,
    error,
    loading,
  };
};

interface StopDeploymentProps {
  envId: string;
  deploymentId: string;
}

export const stopDeployment = ({
  envId,
  deploymentId,
}: StopDeploymentProps): Promise<void> => {
  return api.post(`/v1/deployments/${deploymentId}/stop`, { envId });
};

interface FetchDeploymentProps {
  deploymentId?: string;
  restartToken?: number;
  refreshApp: (n: number) => void;
}

export const useFetchDeployment = ({
  deploymentId,
  restartToken,
  refreshApp,
}: FetchDeploymentProps) => {
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(true);
  const [deploy, setDeploy] = useState<DeploymentV2>();
  const [time, setTime] = useState<number>();

  useEffect(() => {
    let unmounted = false;

    if (!deploymentId) {
      return;
    }

    // Run setLoading only when time is null, subsequent calls shouldn't
    // put the whole view in loading state.
    if (time === null) {
      setLoading(true);
    }

    api
      .fetch<{ deployments: DeploymentV2[] }>(
        `/my/deployments?deploymentId=${deploymentId}`,
      )
      .then(res => {
        const deployment = res.deployments[0];

        if (!deployment) {
          return;
        }

        if (unmounted !== true) {
          setDeploy(deployment);
        }

        if (time && time > 0 && deployment.status === "success") {
          refreshApp(Date.now());
        }

        // If the deployment is still running, then refetch it every 5 seconds
        // by refreshing the time to trigger a new call.
        setTimeout(() => {
          if (unmounted) {
            return;
          }

          setTime(deployment.status === "running" ? Date.now() : 0);
        }, 5000);
      })
      .catch(() => {
        if (unmounted !== true) {
          setError(
            "Something went wrong on our side while fetching deployments. Please try again and if the problem persists contact us from Discord or email.",
          );
        }
      })
      .finally(() => {
        setLoading(false);
      });

    return () => {
      unmounted = true;
    };
  }, [deploymentId, restartToken, time]);

  return { deployment: deploy, loading, error };
};
