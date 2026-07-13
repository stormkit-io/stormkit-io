import nock from "nock";

const endpoint = process.env.API_DOMAIN || "";

interface FetchRepositoriesProps {
  installationId: string;
  search?: string;
  page?: number;
  status?: number;
  response: {
    hasNextPage?: boolean;
    repos: { name: string; fullName: string }[];
  };
}

export const mockFetchRepositories = ({
  installationId,
  search,
  page,
  status = 200,
  response,
}: FetchRepositoriesProps) => {
  return nock(endpoint)
    .get(
      `/provider/github/repos?search=${search}&page=${page}&installationId=${installationId}`
    )
    .reply(status, response);
};

interface FetchInstallationsProps {
  page?: number;
  status?: number;
  response: {
    accounts: { id: string; login: string; avatarUrl: string }[];
  };
}

export const mockFetchInstallations = ({
  status = 200,
  response,
}: FetchInstallationsProps) => {
  return nock(endpoint)
    .get(`/provider/github/accounts`)
    .reply(status, response);
};
