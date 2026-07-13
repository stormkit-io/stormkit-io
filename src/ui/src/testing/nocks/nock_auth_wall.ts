import nock from "nock";

const endpoint = process.env.API_DOMAIN || "";

interface MockFetchAuthWallConfigProps {
  appId: string;
  envId: string;
  response: { authwall: "dev" | "all" | "" };
  status?: number;
}

export const mockFetchAuthWallConfig = ({
  envId,
  appId,
  status = 200,
  response,
}: MockFetchAuthWallConfigProps) =>
  nock(endpoint)
    .get(`/auth-wall/config?appId=${appId}&envId=${envId}`)
    .reply(status, response);

interface MockUpdateAuthWallConfigProps {
  appId: string;
  envId: string;
  authwall: "dev" | "all" | "";
  response?: { ok: boolean };
  status?: number;
}

export const mockUpdateAuthWallConfig = ({
  envId,
  appId,
  authwall,
  status = 200,
  response = { ok: true },
}: MockUpdateAuthWallConfigProps) =>
  nock(endpoint)
    .post("/auth-wall/config", { appId, envId, authwall })
    .reply(status, response);

interface MockCreateNewLoginProps {
  appId: string;
  envId: string;
  email: string;
  password: string;
  response?: { ok: boolean };
  status?: number;
}

export const mockCreateNewLogin = ({
  envId,
  appId,
  email,
  password,
  status = 200,
  response = { ok: true },
}: MockCreateNewLoginProps) =>
  nock(endpoint)
    .post("/auth-wall", { appId, envId, email, password })
    .reply(status, response);

interface MockFetchLoginsProps {
  appId: string;
  envId: string;
  status?: number;
  response: { logins: { email: string; lastLogin: number; id: string }[] };
}

export const mockFetchLogins = ({
  appId,
  envId,
  status,
  response,
}: MockFetchLoginsProps) =>
  nock(endpoint)
    .get(`/auth-wall?appId=${appId}&envId=${envId}`)
    .reply(status, response);

interface MockDeleteLoginsProps {
  appId: string;
  envId: string;
  loginIds?: string;
}

export const mockDeleteLogins = ({
  appId,
  envId,
  loginIds,
}: MockDeleteLoginsProps) =>
  nock(endpoint)
    .delete(`/auth-wall?appId=${appId}&envId=${envId}&id=${loginIds}`)
    .reply(200, { ok: true });
