import nock from "nock";

const endpoint = process.env.API_DOMAIN || "";

export const mockStatsData = {
  totalApps: {
    total: 25,
    new: 5,
    deleted: 2,
  },
  totalDeployments: {
    total: 150,
    current: 30,
    previous: 25,
  },
  avgDeploymentDuration: {
    current: 125.5,
    previous: 170.2,
  },
  totalRequests: {
    current: 10000,
    previous: 8500,
  },
};

export const mockTopDomainsData = [
  {
    id: "1",
    domainName: "example.com",
    current: 15000,
    previous: 12000,
  },
  {
    id: "2",
    domainName: "test.com",
    current: 8500,
    previous: 9000,
  },
  {
    id: "3",
    domainName: "demo.org",
    current: 5200,
    previous: 5200,
  },
];

interface FetchTeamStatsProps {
  teamId?: string;
  status?: number;
  response?: typeof mockStatsData;
}

export const mockFetchTeamStats = ({
  teamId,
  status = 200,
  response = mockStatsData,
}: FetchTeamStatsProps) => {
  return nock(endpoint)
    .get(`/team/stats?teamId=${teamId}`)
    .reply(status, response);
};

interface FetchTopDomainsProps {
  teamId?: string;
  status?: number;
  response?: { domains: typeof mockTopDomainsData };
}

export const mockFetchTopDomains = ({
  teamId,
  status = 200,
  response = { domains: mockTopDomainsData },
}: FetchTopDomainsProps) => {
  return nock(endpoint)
    .get(`/team/stats/domains?teamId=${teamId}`)
    .reply(status, response);
};
