import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

export interface InstanceMailer {
  host: string;
  port: string;
  username: string;
  // Never the real password: a placeholder marking that one is stored.
  password: string;
  from: string;
}

interface FetchInstanceMailerProps {
  refreshToken?: number;
}

export const useFetchInstanceMailer = ({
  refreshToken,
}: FetchInstanceMailerProps) => {
  const [mailer, setMailer] = useState<InstanceMailer | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();

  useEffect(() => {
    setLoading(true);
    setError(undefined);

    api
      .fetch<{ mailer: InstanceMailer | null }>("/admin/mailer")
      .then(({ mailer }) => {
        setMailer(mailer);
      })
      .catch(() => {
        setError("Something went wrong while fetching the default mailer.");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [refreshToken]);

  return { mailer, loading, error };
};
