import nock from "nock";

const endpoint = process.env.API_DOMAIN || "localhost";

interface MockDomainsFetchProps {
  appId: string;
  envId: string;
  domainName?: string;
  afterId?: string;
  verified?: boolean;
  status?: number;
  response: { domains: Domain[] };
}

export const mockFetchDomains = ({
  appId,
  envId,
  status,
  afterId,
  verified,
  response,
  domainName,
}: MockDomainsFetchProps) => {
  const qs = new URLSearchParams(
    JSON.parse(
      JSON.stringify({
        appId,
        envId,
        verified,
        afterId,
        domainName,
        pageSize: 100,
      })
    )
  );

  return nock(endpoint)
    .get(`/v1/domains?${qs.toString()}`)
    .reply(status, response);
};

interface MockDomainInsertProps {
  appId: string;
  envId: string;
  domain: string;
  status?: number;
  response?: { ok?: boolean; error?: string };
}

export const mockDomainInsert = ({
  appId,
  envId,
  domain,
  status = 200,
  response = { ok: true },
}: MockDomainInsertProps) =>
  nock(endpoint)
    .post(`/v1/domains`, { appId, envId, domain })
    .reply(status, response);

interface MockFetchDomainsInfoProps {
  appId: string;
  envId: string;
  domainId: string;
  status?: number;
  response: any;
}

export const mockFetchDomainsInfo = ({
  appId,
  envId,
  domainId,
  status = 200,
  response,
}: MockFetchDomainsInfoProps) =>
  nock(endpoint)
    .get(`/domains/lookup?appId=${appId}&envId=${envId}&domainId=${domainId}`)
    .reply(status, response);

interface MockDeleteDomainProps {
  appId: string;
  envId: string;
  domainId: string;
  status?: number;
  response: { ok: boolean };
}

export const mockDeleteDomain = ({
  appId,
  envId,
  domainId,
  status = 200,
  response = { ok: true },
}: MockDeleteDomainProps) => {
  return nock(endpoint)
    .delete(`/v1/domains?appId=${appId}&domainId=${domainId}&envId=${envId}`)
    .reply(status, response);
};

interface MockUpdateDomainProps {
  appId: string;
  envId: string;
  domainId: string;
  analyticsExcluded: boolean;
  status?: number;
  response?: { ok?: boolean; error?: string };
}

export const mockUpdateDomain = ({
  appId,
  envId,
  domainId,
  analyticsExcluded,
  status = 200,
  response = { ok: true },
}: MockUpdateDomainProps) => {
  return nock(endpoint)
    .put("/v1/domains", { appId, envId, domainId, analyticsExcluded })
    .reply(status, response);
};

interface UpdateCustomCertProps {
  appId: string;
  envId: string;
  domainId: string;
  status: number;
  certKey: string;
  certValue: string;
  response: { ok: boolean };
}

export const mockUpdateCustomCert = ({
  appId,
  envId,
  domainId,
  certKey,
  certValue,
  status = 200,
  response = { ok: true },
}: UpdateCustomCertProps) => {
  return nock(endpoint)
    .put("/v1/domains/cert", { appId, envId, domainId, certKey, certValue })
    .reply(status, response);
};

interface DeleteCustomCertProps {
  appId: string;
  envId: string;
  domainId: string;
  status: number;
  response: { ok: boolean };
}

export const mockDeleteCustomCert = ({
  appId,
  envId,
  domainId,
  status = 200,
  response = { ok: true },
}: DeleteCustomCertProps) => {
  return nock(endpoint)
    .delete(`/v1/domains/cert?appId=${appId}&envId=${envId}&domainId=${domainId}`)
    .reply(status, response);
};
