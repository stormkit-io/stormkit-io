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
  migrationsEnabled?: boolean;
  migrationsFolder?: string;
}

interface UseFetchSchemaProps {
  envId: string;
  refreshToken?: number;
}

export const useFetchSchema = ({
  envId,
  refreshToken,
}: UseFetchSchemaProps) => {
  const [schema, setSchema] = useState<Schema | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();

  useEffect(() => {
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
  }, [envId, refreshToken]);

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
}

export const updateSchema = ({
  appId,
  envId,
  migrationsFolder,
  migrationsEnabled,
}: UpdateSchemaProps) => {
  return api.post(`/schema/configure`, {
    appId,
    envId,
    migrationsFolder,
    migrationsEnabled,
  });
};
