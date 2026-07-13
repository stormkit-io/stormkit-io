import qs from "query-string";
import nock from "nock";

const endpoint = process.env.API_DOMAIN || "";

interface FetchRepositoriesProps {
  query: {
    membership?: "true";
    order_by?: "id";
    per_page: number;
    page: number;
  };
  status?: number;
  response: { path_with_namespace: string; name: string }[];
}

export const mockFetchRepositories = ({
  query,
  status = 200,
  response,
}: FetchRepositoriesProps) => {
  const params = qs.stringify({
    membership: "true",
    order_by: "id",
    ...query,
  });

  return nock(endpoint).get(`/projects?${params}`).reply(status, response);
};
