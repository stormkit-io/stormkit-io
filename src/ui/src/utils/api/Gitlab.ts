import qs from "query-string";
import { prepareHeaders, errTokenExpired } from "./helpers";
import { LocalStorage } from "~/utils/storage";
import { LS_ACCESS_TOKEN } from "./Api";

interface RepositoriesProps {
  page?: number;
  size?: number;
}

interface Repository {
  name: string;
  path_with_namespace: string;
}

interface RepositoriesResponse {
  nextPage: string;
  repos: Repository[];
}

class Gitlab {
  baseurl = "https://gitlab.com/api/v4";

  // This is the access token required to fetch repositories.
  accessToken = LocalStorage.get(LS_ACCESS_TOKEN) as string;

  /**
   * User returns the currently logged in user.
   * We need the user-to-server token to get a valid response
   * from this endpoint. This token can be obtained by using
   * the Auth.context:loginOauth method.
   *
   * @param {*} token The access token to make api calls.
   */
  user() {
    return new Promise((resolve, reject) => {
      const headers = prepareHeaders(this.accessToken);
      const request = new Request(`${this.baseurl}/user`, { headers });

      return fetch(request).then(res => {
        if (res.status === 401) {
          return reject(errTokenExpired);
        }

        return res.json().then(user => {
          resolve(user);
        });
      });
    });
  }

  repositories({
    page = 1,
    size = 20,
  }: RepositoriesProps = {}): Promise<RepositoriesResponse> {
    return new Promise((resolve, reject) => {
      const headers = prepareHeaders(this.accessToken);
      const params = {
        membership: "true",
        order_by: "id",
        per_page: size,
        page,
      };

      const url = `${this.baseurl}/projects?${qs.stringify(params)}`;
      const request = new Request(url, { headers });

      return fetch(request).then(res => {
        if (res.status === 401) {
          return reject(errTokenExpired);
        }

        return res.json().then(json => {
          resolve({
            repos: json,
            nextPage: res.headers.get("X-Next-Page") || "",
          });
        });
      });
    });
  }
}

export default new Gitlab();
