import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

interface Table {
  name: string;
  rows: number; // estimated number of rows
  size: number; // estimated size in bytes
}

export interface Schema {
  name: string;
  tables: Table[];
  injectEnvVars?: boolean;
  migrationsEnabled?: boolean;
  migrationsFolder?: string;
}

interface UseFetchSchemaProps {
  envId: string;
  refreshToken?: number;
  isCloud?: boolean;
}

export const useFetchSchema = ({
  envId,
  refreshToken,
  isCloud,
}: UseFetchSchemaProps) => {
  const [schema, setSchema] = useState<Schema | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();

  useEffect(() => {
    if (isCloud) {
      setLoading(false);
      return;
    }

    api
      .fetch<{ schema: Schema | null }>(`/schema?envId=${envId}`)
      .then(({ schema }) => {
        setSchema(schema);
      })
      .catch(() => {
        setError("Unknown error while fetching database.");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [envId, refreshToken, isCloud]);

  return { schema, loading, error };
};

interface CreateSchemaProps {
  appId: string;
  envId: string;
}

export const createSchema = ({ appId, envId }: CreateSchemaProps) => {
  return api.post<{ schema: string }>(`/schema`, {
    appId,
    envId,
  });
};

interface DeleteSchemaProps {
  envId: string;
  appId: string;
}

export const deleteSchema = ({ appId, envId }: DeleteSchemaProps) => {
  return api.delete(`/schema?appId=${appId}&envId=${envId}`);
};

interface UpdateSchemaProps {
  appId: string;
  envId: string;
  migrationsFolder: string;
  migrationsEnabled: boolean;
  injectEnvVars: boolean;
}

export const updateSchema = ({
  appId,
  envId,
  migrationsFolder,
  migrationsEnabled,
  injectEnvVars,
}: UpdateSchemaProps) => {
  return api.post(`/schema/configure`, {
    appId,
    envId,
    migrationsFolder,
    migrationsEnabled,
    injectEnvVars,
  });
};
