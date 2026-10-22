import nock from "nock";

const endpoint = process.env.API_DOMAIN || "";

interface MockFetchAuthWallConfigProps {
  appId: string;
  envId: string;
  response: { authwall: "dev" | "all" | "" };
  status?: number;
  delay?: number;
}

export const mockFetchAuthWallConfig = ({
  envId,
  appId,
  status = 200,
  response,
  delay = 0,
}: MockFetchAuthWallConfigProps) =>
  nock(endpoint)
    .get(`/auth-wall/config?appId=${appId}&envId=${envId}`)
    .delay(delay)
    .reply(status, response);
