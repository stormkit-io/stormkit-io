import nock from "nock";
import type {
  AuthWallMode,
  TeamAuthWallApp,
  TeamAuthWallLogin,
} from "~/pages/team/auth-wall/actions";

const endpoint = process.env.API_DOMAIN || "";

interface MockFetchTeamEnvsProps {
  teamId: string;
  apps?: TeamAuthWallApp[];
  status?: number;
  delay?: number;
}

export const mockFetchTeamEnvs = ({
  teamId,
  apps = [],
  status = 200,
  delay = 0,
}: MockFetchTeamEnvsProps) =>
  nock(endpoint)
    .get(`/team/auth-wall/envs?teamId=${teamId}`)
    .delay(delay)
    .reply(status, { apps });

interface MockFetchTeamLoginsProps {
  teamId: string;
  from?: number;
  logins?: TeamAuthWallLogin[];
  hasNextPage?: boolean;
  status?: number;
}

export const mockFetchTeamLogins = ({
  teamId,
  from = 0,
  logins = [],
  hasNextPage = false,
  status = 200,
}: MockFetchTeamLoginsProps) =>
  nock(endpoint)
    .get(`/team/auth-wall/logins?teamId=${teamId}&from=${from}`)
    .reply(status, { logins, hasNextPage });

interface MockCreateTeamLoginProps {
  body: Record<string, unknown>;
  status?: number;
  response?: Record<string, unknown>;
}

export const mockCreateTeamLogin = ({
  body,
  status = 201,
  response = {},
}: MockCreateTeamLoginProps) =>
  nock(endpoint).post("/team/auth-wall/logins", body).reply(status, response);

export const mockUpdateTeamLogin = ({
  body,
  status = 200,
  response = {},
}: MockCreateTeamLoginProps) =>
  nock(endpoint).patch("/team/auth-wall/logins", body).reply(status, response);

interface MockDeleteTeamLoginsProps {
  teamId: string;
  loginIds: string[];
}

export const mockDeleteTeamLogins = ({
  teamId,
  loginIds,
}: MockDeleteTeamLoginsProps) =>
  nock(endpoint)
    .delete(`/team/auth-wall/logins?teamId=${teamId}&id=${loginIds.join(",")}`)
    .reply(200, { ok: true });

interface MockUpdateEnvAuthWallProps {
  teamId: string;
  envId: string;
  authWall: AuthWallMode;
  status?: number;
}

export const mockUpdateEnvAuthWall = ({
  teamId,
  envId,
  authWall,
  status = 200,
}: MockUpdateEnvAuthWallProps) =>
  nock(endpoint)
    .put("/team/auth-wall/envs", { teamId, envId, authWall })
    .reply(status, { ok: true });
