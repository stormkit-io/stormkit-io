import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

interface MailerConfig {
  host: string;
  port: string;
  username: string;
  password: string;
}

interface FetchMailerConfigProps {
  appId: string;
  envId: string;
  refreshToken?: number;
}

export const useFetchMailerConfig = ({
  envId,
  appId,
  refreshToken,
}: FetchMailerConfigProps) => {
  const [config, setConfig] = useState<MailerConfig>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();

  useEffect(() => {
    setLoading(true);
    setError(undefined);

    api
      .fetch<{ config: MailerConfig }>(
        `/mailer/config?appId=${appId}&envId=${envId}`,
      )
      .then(({ config }) => {
        setConfig(config);
      })
      .catch(() => {
        setError("Something went wrong while fetching mailer config.");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [refreshToken, appId, envId]);

  return { config, loading, error };
};
