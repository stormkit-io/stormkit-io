import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

export interface Database {
  id: string;
  name: string;
  type: "provisioned" | "external";
  connectionString?: string;
  createdAt: number;
}

interface UseFetchDatabaseProps {
  envId: string;
  refreshToken?: number;
}

export const useFetchDatabase = ({
  envId,
  refreshToken,
}: UseFetchDatabaseProps) => {
  const [database, setDatabase] = useState<Database | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();

  useEffect(() => {
    api
      .fetch<{ database: Database | null }>(`/database?envId=${envId}`)
      .then(({ database }) => {
        setDatabase(database);
      })
      .catch(() => {
        setError("Unknown error while fetching database.");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [envId, refreshToken]);

  return { database, loading, error };
};
