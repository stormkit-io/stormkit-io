import nock from "nock";
import type { Schema } from "~/pages/apps/[id]/environments/[env-id]/database/actions";

const endpoint = process.env.API_DOMAIN || "";

interface MockFetchSchemaProps {
  envId: string;
  response?: { schema: Schema | null };
  status?: number;
}

export const mockFetchSchema = ({
  envId,
  status = 200,
  response = { schema: null },
}: MockFetchSchemaProps) => {
  return nock(endpoint).get(`/schema?envId=${envId}`).reply(status, response);
};

interface MockCreateSchemaProps {
  payload: {
    appId: string;
    envId: string;
  };
  response?: { schema: string };
  status?: number;
}

export const mockCreateSchema = ({
  payload,
  status = 200,
  response = { schema: `a${payload.appId}e${payload.envId}` },
}: MockCreateSchemaProps) => {
  return nock(endpoint).post("/schema", payload).reply(status, response);
};

interface MockUpdateSchemaConfigProps {
  payload: {
    appId: string;
    envId: string;
    migrationsEnabled: boolean;
    migrationsPath: string;
  };
  status?: number;
}

export const mockUpdateSchemaConfig = ({
  payload,
  status = 200,
}: MockUpdateSchemaConfigProps) => {
  return nock(endpoint).post("/schema/configure", payload).reply(status, {});
};
