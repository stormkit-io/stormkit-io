import nock from "nock";

const endpoint = process.env.API_DOMAIN || "";

interface PlaygroundProps {
  appId: string;
  envId: string;
  redirects: any;
  address: string;
  status?: number;
  response: {
    match: boolean;
    status: number;
    redirect: string;
    rewrite: string;
    proxy: boolean;
  };
}

export const mockPlayground = ({
  appId,
  envId,
  address,
  redirects,
  response,
  status = 200,
}: PlaygroundProps) =>
  nock(endpoint)
    .post("/redirects/playground", { appId, envId, redirects, address })
    .reply(status, response);
