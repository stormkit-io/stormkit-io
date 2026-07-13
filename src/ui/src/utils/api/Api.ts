import { LocalStorage } from "~/utils/storage";

type Body =
  | string
  | Blob
  | ArrayBufferView
  | ArrayBuffer
  | FormData
  | URLSearchParams
  | ReadableStream<Uint8Array>
  | null
  | undefined;

interface Options {
  baseurl: string;
}

interface FetchOptions extends Record<string, unknown> {
  method?: "POST" | "GET" | "PUT" | "DELETE" | "PATCH";
  headers?: Record<string, string>;
  body?: Body;
  params?: Body; // alias for body
}

export const LS_ACCESS_TOKEN = "skit_access_token";
export const LS_PROVIDER = "skit_provider";
export const LS_TOKEN_KEY = "skit_token";

class Api {
  static isAbsolute(url: string): boolean {
    return url?.match(/^(https?|\/\/)/) !== null;
  }

  baseurl = "";
  headers: Record<string, string> = {};

  constructor(opts: Options) {
    this.baseurl = opts.baseurl;
    this.headers = {
      "Content-Type": "application/json",
    };

    const token = LocalStorage.get<string>(LS_TOKEN_KEY);

    if (token) {
      this.setAuthToken(token);
    }
  }

  setAuthToken(token: string): void {
    LocalStorage.set(LS_TOKEN_KEY, token);
    this.headers.Authorization = "Bearer " + token;
  }

  getAuthToken(): string | undefined {
    return LocalStorage.get(LS_TOKEN_KEY);
  }

  removeAuthToken(): void {
    LocalStorage.del(LS_TOKEN_KEY);
    delete this.headers.Authorization;
  }

  getHeaders(additional: Record<string, string> = {}): Headers {
    const headers = new Headers();

    Object.keys(this.headers).forEach(k => {
      headers.append(k, this.headers[k]);
    });

    Object.keys(additional).forEach(k => {
      headers.append(k, additional[k]);
    });

    return headers;
  }

  private async handle403(response: Response): Promise<void> {
    try {
      const json = await response.json();

      if (json?.user === false && window.location.pathname !== "/auth") {
        window.location.href = "/auth";
        return;
      }
    } catch {
      throw response;
    }
  }

  /**
   * Wrapper function for the fetch API.
   *
   * @param url
   * @param opts
   * @return {Promise<any>}
   */
  async fetch<T>(url: string, opts: FetchOptions = {}): Promise<T> {
    // Get the baseurl from the headers if the baseurl is not set
    if (this.baseurl === "") {
      const resp = await fetch(
        new Request(window.location.href, { method: "HEAD" })
      );

      this.baseurl = resp.headers.get("x-sk-api") || "";
    }

    if (Api.isAbsolute(url) === false) {
      url = this.baseurl.replace(/\/+$/, "") + "/" + url.replace(/^\//, "");
    }

    const headers = this.getHeaders(opts.headers);

    // https://stackoverflow.com/a/49510941/1075534
    if (opts.body instanceof FormData) {
      headers.delete("Content-Type");
    }

    // @ts-ignore
    const request = new Request(url, { ...opts, headers });

    try {
      const resp = await fetch(request);

      if (resp.status === 403) {
        this.handle403(resp);
      }

      if (resp.status.toString()[0] !== "2") {
        throw resp;
      }

      if (resp.headers.get("content-type") != "application/json") {
        return resp as T;
      }

      try {
        const json = await resp.json();

        if (json && json.jwt) {
          this.setAuthToken(json.jwt);
        }

        return json as T;
      } catch {
        if (resp.status === 201) {
          return {} as T;
        }
      }

      return {} as T;
    } catch (e) {
      throw e;
    }
  }

  // Helper method for sending patch requests.
  async patch<T>(url: string, opts: FetchOptions = {}): Promise<T> {
    // This line of code allows using post as patch(url, myRequestBody)
    if (!opts.body && !opts.params) {
      opts = { body: opts as unknown as Body };
    }

    opts.method = "PATCH";

    if (opts.body || opts.params) {
      opts.body = JSON.stringify(opts.body || opts.params);
    }

    return this.fetch(url, opts);
  }

  // Helper method for sending post requests.
  async post<T>(url: string, opts: FetchOptions = {}): Promise<T> {
    // This line of code allows using post as post(url, myRequestBody)
    if (!opts.body && !opts.params) {
      opts = { body: opts as unknown as Body };
    }

    opts.method = "POST";

    if (typeof opts.params !== "string" && opts.params) {
      opts.body = JSON.stringify(opts.params);
    }

    if (typeof opts.body !== "string") {
      opts.body = JSON.stringify(opts.body);
    }

    return this.fetch(url, opts);
  }

  async upload<T>(url: string, opts: FetchOptions = {}): Promise<T> {
    opts.method = "POST";

    if (!opts.body) {
      return Promise.reject("body is a required parameter");
    }

    return this.fetch(url, opts);
  }

  // Helper method for sending put requests.
  async put<T>(url: string, opts: FetchOptions = {}): Promise<T> {
    // This line of code allows using post as put(url, myRequestBody)
    if (!opts.body && !opts.params) {
      opts = { body: opts as unknown as Body };
    }

    opts.method = "PUT";

    if (opts.body || opts.params) {
      opts.body = JSON.stringify(opts.body || opts.params);
    }

    return this.fetch(url, opts);
  }

  // Helper method for sending delete requests.
  async delete<T>(url: string, opts: FetchOptions = {}): Promise<T> {
    // This line of code allows using post as put(url, myRequestBody)
    if (!opts.body && !opts.params) {
      opts = { body: opts as unknown as Body };
    }

    opts.method = "DELETE";

    if (opts.body || opts.params) {
      opts.body = JSON.stringify(opts.body || opts.params);
    }

    return this.fetch(url, opts);
  }

  async errors(res: Response): Promise<Array<string>> {
    const unknown = [
      "Something went wrong while performing operation. Please retry, if the problem persists reach us out on Discord or through email.",
    ];

    try {
      const data = await res.json();

      if (data.error) {
        return [data.error];
      } else if (data.errors) {
        return Object.values(data.errors);
      } else {
        return unknown;
      }
    } catch {
      return unknown;
    }
  }
}

export default new Api({
  baseurl: process.env.API_DOMAIN || "",
});
