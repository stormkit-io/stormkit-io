import qs from "query-string";
import { prepareHeaders, errTokenExpired } from "./helpers";
import { LocalStorage } from "~/utils/storage";
import { LS_ACCESS_TOKEN } from "./Api";

interface RepositoriesProps {
  team?: string;
  params?: {
    role?: "admin" | "member";
    pagelen?: number;
  };
}

interface Repository {
  full_name: string;
  name: string;
  type: "repository";
}

interface RepositoriesResponse {
  next: boolean;
  values: Repository[];
}

class Bitbucket {
  baseurl = "https://api.bitbucket.org/2.0";

  // This is the access token required to fetch repositories.
  accessToken = LocalStorage.get(LS_ACCESS_TOKEN) as string;

  user() {
    return new Promise((resolve, reject) => {
      const headers = prepareHeaders(this.accessToken);
      const request = new Request(this.baseurl + "/user", { headers });

      return fetch(request).then(res => {
        if (res.status === 401) {
          return reject(errTokenExpired);
        }

        resolve(res.json());
      });
    });
  }

  repositories({
    team,
    params = {},
  }: RepositoriesProps = {}): Promise<RepositoriesResponse> {
    const url = team ? `/repositories/${team}` : "/repositories";

    return new Promise((resolve, reject) => {
      const headers = prepareHeaders(this.accessToken);

      // push default params if no params set yet
      if (Object.keys(params).length === 0) {
        params.role = "admin";
        params.pagelen = 100;
      }

      // build query
      const query = qs.stringify(params);

      const request = new Request(`${this.baseurl}${url}?${query}`, {
        headers,
      });

      return fetch(request).then(res => {
        if (res.status === 401) {
          return reject(errTokenExpired);
        }

        resolve(res.json());
      });
    });
  }
}

export default new Bitbucket();
