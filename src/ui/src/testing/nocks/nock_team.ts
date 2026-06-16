import nock from "nock";

const endpoint = process.env.API_DOMAIN || "";

interface MockFetchTeamProps {
  status?: number;
  response: Team[];
}

export const mockFetchTeam = ({ status = 200, response }: MockFetchTeamProps) =>
  nock(endpoint).get("/v1/teams").reply(status, { teams: response });

interface MockCreateTeamProps {
  name: string;
  status?: number;
  response: Team;
}

export const mockCreateTeam = ({
  name,
  status = 200,
  response,
}: MockCreateTeamProps) =>
  nock(endpoint).post("/v1/teams", { name }).reply(status, { team: response });

interface MockUpdateTeamProps {
  teamId: string;
  name: string;
  status?: number;
  response: Team;
}

export const mockUpdateTeam = ({
  name,
  teamId,
  status = 200,
  response,
}: MockUpdateTeamProps) =>
  nock(endpoint).patch("/team", { name, teamId }).reply(status, response);

interface MockRemoveTeamProps {
  teamId: string;
  status?: number;
  response?: { ok: boolean };
}

export const mockRemoveTeam = ({
  teamId,
  status = 200,
  response,
}: MockRemoveTeamProps) =>
  nock(endpoint).delete(`/team?teamId=${teamId}`).reply(status, response);

interface MockMigrateTeamProps {
  teamId: string;
  appId: string;
  status?: number;
  response?: { ok: boolean };
}

export const mockMigrateApp = ({
  teamId,
  appId,
  status = 200,
  response = { ok: true },
}: MockMigrateTeamProps) =>
  nock(endpoint)
    .post("/team/migrate", { teamId, appId })
    .reply(status, response);
