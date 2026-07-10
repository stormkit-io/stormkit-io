import nock from "nock";

const endpoint = process.env.API_DOMAIN || "";

interface MockFetchMaintenanceConfigProps {
  appId: string;
  envId: string;
  response: { maintenance: "on" | "" };
  status?: number;
}

export const mockFetchMaintenanceConfig = ({
  envId,
  appId,
  status = 200,
  response,
}: MockFetchMaintenanceConfigProps) =>
  nock(endpoint)
    .get(`/maintenance/config?appId=${appId}&envId=${envId}`)
    .reply(status, response);

interface MockUpdateMaintenanceConfigProps {
  appId: string;
  envId: string;
  maintenance: "on" | "";
  response?: { ok: boolean };
  status?: number;
}

export const mockUpdateMaintenanceConfig = ({
  envId,
  appId,
  maintenance,
  status = 200,
  response = { ok: true },
}: MockUpdateMaintenanceConfigProps) =>
  nock(endpoint)
    .post("/maintenance/config", { appId, envId, maintenance })
    .reply(status, response);
