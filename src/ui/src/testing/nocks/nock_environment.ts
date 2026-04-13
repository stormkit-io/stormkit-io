import nock from "nock";
import { expect } from "vitest";

const endpoint = process.env.API_DOMAIN || "";

// Used for POST /app/env (insert).
const toRequestObject = (environment: Environment) => {
  return JSON.parse(
    JSON.stringify({
      id: environment.id,
      appId: environment.appId,
      env: environment.env,
      branch: environment.branch,
      autoPublish: environment.autoPublish,
      autoDeploy: environment.autoDeploy,
      autoDeployBranches: environment.autoDeployBranches,
      build: {
        statusChecks: environment.build.statusChecks,
        previewLinks: environment.build.previewLinks,
        apiFolder: environment.build.apiFolder,
        headers: environment.build.headers || "",
        headersFile: environment.build.headersFile,
        errorFile: environment.build.errorFile,
        redirectsFile: environment.build.redirectsFile,
        redirects: environment.build.redirects,
        distFolder: environment.build.distFolder,
        installCmd: environment.build.installCmd || "",
        buildCmd: environment.build.buildCmd || "",
        serverCmd: environment.build.serverCmd || "",
        vars: environment.build.vars,
      },
    }),
  );
};

// Used for PUT /v1/env (update) — flat body, no nesting.
const toUpdateRequestObject = (environment: Environment) => {
  return JSON.parse(
    JSON.stringify({
      envId: environment.id,
      name: environment.name || environment.env,
      branch: environment.branch,
      autoPublish: !!environment.autoPublish,
      autoDeploy: !!environment.autoDeploy,
      autoDeployBranches: environment.autoDeployBranches || undefined,
      autoDeployCommits: environment.autoDeployCommits || undefined,
      buildCmd: environment.build.buildCmd?.trim() || "",
      serverCmd: environment.build.serverCmd?.trim() || "",
      installCmd: environment.build.installCmd?.trim() || "",
      distFolder: (
        environment.build.distFolder ||
        environment.build.serverFolder ||
        ""
      ).trim(),
      headers: environment.build.headers?.trim() || "",
      headersFile: environment.build.headersFile,
      redirectsFile: environment.build.redirectsFile,
      errorFile: environment.build.errorFile,
      apiFolder: environment.build.apiFolder,
      apiPathPrefix: environment.build.apiPathPrefix,
      previewLinks: environment.build.previewLinks !== false,
      statusChecks: environment.build.statusChecks,
      redirects: environment.build.redirects,
      envVars: environment.build.vars,
    }),
  );
};

interface FetchStatusProps {
  url: string;
  appId: string;
  status?: number;
  response?: { status: number };
}

export const mockFetchStatus = ({
  appId,
  url,
  status = 200,
  response = { status: 200 },
}: FetchStatusProps) =>
  nock(endpoint).post(`/app/proxy`, { appId, url }).reply(status, response);

interface FetchEnvironmentsProps {
  app: App;
  status?: number;
  response?: object;
}

export const mockFetchEnvironments = ({
  app,
  status,
  response,
}: FetchEnvironmentsProps) =>
  nock(endpoint).get(`/v1/envs?appId=${app.id}`).reply(status, response);

interface InsertEnvironmentProps {
  environment: Environment;
  status?: number;
  response?: { envId?: string; error?: string };
}

export const mockInsertEnvironment = ({
  environment,
  status = 200,
  response = { envId: environment.id },
}: InsertEnvironmentProps) => {
  const data = toRequestObject(environment);
  delete data.id;

  return nock(endpoint)
    .post(`/app/env`, (body: any) => {
      expect(body).toEqual(data);
      return true;
    })
    .reply(status, response);
};

interface UpdateEnvironmentProps {
  environment: Environment;
  status?: number;
  response?: { ok: true };
}

export const mockUpdateEnvironment = ({
  environment,
  status = 200,
  response = { ok: true },
}: UpdateEnvironmentProps) => {
  return nock(endpoint)
    .put(`/v1/env`, (body: any) => {
      expect(body).toEqual(toUpdateRequestObject(environment));
      return true;
    })
    .reply(status, response);
};

interface DeleteEnvironmentProps {
  appId: string;
  env: string;
  status?: number;
  response?: { ok: boolean };
}

export const mockDeleteEnvironment = ({
  appId,
  env,
  status = 200,
  response = { ok: true },
}: DeleteEnvironmentProps) =>
  nock(endpoint)
    .delete(`/app/env`, {
      appId,
      env,
    })
    .reply(status, response);
