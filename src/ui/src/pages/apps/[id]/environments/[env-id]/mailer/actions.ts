import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

interface MailerConfig {
  host: string;
  port: string;
  username: string;
  // Never the real password: a placeholder marking that one is stored.
  password: string;
}

export interface Email {
  id: string;
  envId: string;
  from: string;
  to: string;
  subject: string;
  body: string;
  sentAt: number;
}

interface FetchMailerConfigProps {
  appId: string;
  envId: string;
  refreshToken?: number;
}

interface FetchEmailsProps {
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
        `/v1/mailer/config?appId=${appId}&envId=${envId}`,
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

export const useFetchEmails = ({
  appId,
  envId,
  refreshToken,
}: FetchEmailsProps) => {
  const [emails, setEmails] = useState<Email[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();

  useEffect(() => {
    let unmounted = false;

    setLoading(true);
    setError(undefined);

    api
      .fetch<{ emails: Email[] }>(`/mailer?appId=${appId}&envId=${envId}`)
      .then(({ emails }) => {
        if (!unmounted) {
          setEmails(emails || []);
        }
      })
      .catch(() => {
        if (!unmounted) {
          setError("Something went wrong while fetching sent emails.");
        }
      })
      .finally(() => {
        if (!unmounted) {
          setLoading(false);
        }
      });

    return () => {
      unmounted = true;
    };
  }, [refreshToken, appId, envId]);

  return { emails, loading, error };
};
