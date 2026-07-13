import qs from "query-string";
import nock from "nock";

const endpoint = process.env.API_DOMAIN || "";

interface FetchRepositoriesProps {
  query: Record<string, unknown>;
  status?: number;
  response: {
    next: string;
    values: { full_name: string; name: string; type: "repository" }[];
  };
}

export const mockFetchRepositories = ({
  query,
  status = 200,
  response,
}: FetchRepositoriesProps) => {
  const params = qs.stringify(query);
  return nock(endpoint).get(`/repositories?${params}`).reply(status, response);
};
