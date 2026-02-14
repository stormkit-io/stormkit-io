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

interface MockUpdateSchemaProps {
  payload: {
    appId: string;
    envId: string;
    migrationsEnabled: boolean;
    migrationsFolder: string;
    injectEnvVars: boolean;
  };
  status?: number;
}

export const mockUpdateSchema = ({
  payload,
  status = 200,
}: MockUpdateSchemaProps) => {
  return nock(endpoint).post("/schema/configure", payload).reply(status, {});
};

interface MockDeleteSchemaProps {
  payload: {
    appId: string;
    envId: string;
  };
  status?: number;
}

export const mockDeleteSchema = ({
  payload,
  status = 200,
}: MockDeleteSchemaProps) => {
  return nock(endpoint)
    .delete(`/schema?envId=${payload.envId}&appId=${payload.appId}`)
    .reply(status, {});
};
