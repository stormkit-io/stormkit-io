import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

export type MaintenanceConfig = "on" | "";

interface UseFetchMaintenanceConfig {
  appId: string;
  envId: string;
  refreshToken?: number;
}

export const useFetchMaintenanceConfig = ({
  appId,
  envId,
  refreshToken,
}: UseFetchMaintenanceConfig) => {
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string>();
  const [config, setConfig] = useState<MaintenanceConfig>();

  useEffect(() => {
    api
      .fetch<{ maintenance: MaintenanceConfig }>(
        `/maintenance/config?appId=${appId}&envId=${envId}`
      )
      .then(({ maintenance }) => {
        setConfig(maintenance);
      })
      .catch(() => {
        setError("Something went wrong while fetching the maintenance config");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [appId, envId, refreshToken]);

  return { loading, error, config };
};

interface UpdateMaintenanceConfigProps {
  appId: string;
  envId: string;
  maintenance: MaintenanceConfig;
}

export const updateMaintenanceConfig = ({
  appId,
  envId,
  maintenance,
}: UpdateMaintenanceConfigProps) => {
  return api.post("/maintenance/config", {
    appId,
    envId,
    maintenance,
  });
};
