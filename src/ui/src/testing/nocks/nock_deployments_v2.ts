import nock from "nock";

const endpoint = process.env.API_DOMAIN || "localhost";

interface MockFetchDeploymentCallProps {
  teamId?: string;
  envId?: string;
  deploymentId?: string;
  status?: number;
  response: { deployments: DeploymentV2[] };
}

export const mockFetchDeployments = ({
  teamId,
  deploymentId,
  envId,
  status = 200,
  response,
}: MockFetchDeploymentCallProps) => {
  const params = new URLSearchParams(
    JSON.parse(JSON.stringify({ teamId, deploymentId, envId }))
  );

  return nock(endpoint)
    .get(`/my/deployments?${params}`)
    .reply(status, response);
};

interface MockStopDeploymentProps {
  envId: string;
  deploymentId: string;
  status?: number;
  response?: object;
}

export const mockStopDeployment = ({
  envId,
  deploymentId,
  status = 200,
  response = { ok: true },
}: MockStopDeploymentProps) =>
  nock(endpoint)
    .post(`/v1/deployments/${deploymentId}/stop`, { envId })
    .reply(status, response);

interface MockDeleteDeploymentProps {
  appId: string;
  deploymentId: string;
  status?: number;
  response?: object;
}

export const mockDeleteDeployment = ({
  appId,
  deploymentId,
  status = 200,
  response = { ok: true },
}: MockDeleteDeploymentProps) =>
  nock(endpoint)
    .delete(`/app/deploy`, { appId, deploymentId })
    .reply(status, response);

interface mockFetchManifestProps {
  appId: string;
  deploymentId: string;
  status?: number;
  response: { manifest?: Manifest };
}

export const mockFetchManifest = ({
  appId,
  deploymentId,
  status = 200,
  response = { manifest: {} },
}: mockFetchManifestProps) =>
  nock(endpoint)
    .get(`/app/${appId}/manifest/${deploymentId}`)
    .reply(status, response);

interface MockPublishDeploymentsProps {
  appId: string;
  envId: string;
  publish: { percentage: number; deploymentId: string }[];
  status?: number;
  response?: object;
}

export const mockPublishDeployments = ({
  appId,
  envId,
  publish,
  status = 200,
  response = { ok: true },
}: MockPublishDeploymentsProps) =>
  nock(endpoint)
    .post(`/app/deployments/publish`, { appId, envId, publish })
    .reply(status, response);
