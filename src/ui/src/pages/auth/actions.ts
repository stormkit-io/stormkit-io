import { useEffect, useState } from "react";
import api, { LS_ACCESS_TOKEN, LS_PROVIDER } from "~/utils/api/Api";
import bitbucketApi from "~/utils/api/Bitbucket";
import gitlabApi from "~/utils/api/Gitlab";
import openPopup, { DataMessage } from "~/utils/helpers/popup";
import { LocalStorage } from "~/utils/storage";

interface FetchUserResponse {
  accounts: Array<ConnectedAccount>;
  user: User;
  metrics?: UserMetrics;
}

export const useFetchUser = () => {
  const token = api.getAuthToken();
  const [error, setError] = useState<string | null>(null);
  const [user, setUser] = useState<User>();
  const [accounts, setAccounts] = useState<Array<ConnectedAccount>>([]);
  const [metrics, setMetrics] = useState<UserMetrics>();
  const [loading, setLoading] = useState<boolean>(!!token);

  useEffect(() => {
    if (!token) {
      return;
    }

    setLoading(true);
    api.setAuthToken(token);
    api
      .fetch<FetchUserResponse>("/user")
      .then(({ user, accounts, metrics }) => {
        if (user) {
          setUser(user);
          setAccounts(accounts);
          setMetrics(metrics);
        }
      })
      .catch(e => {
        if (e.status !== 401) {
          setError("Something went wrong, log in again.");
        }
      })
      .finally(() => {
        setLoading(false);
      });
  }, [api, token]);

  return { error, user, accounts, metrics, loading, setUser };
};

export interface Providers {
  github: boolean;
  gitlab: boolean;
  bitbucket: boolean;
  basicAuth?: "enabled";
}

export const useFetchActiveProviders = () => {
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(true);
  const [providers, setProviders] = useState<Providers>();

  useEffect(() => {
    setLoading(true);
    setError(undefined);

    api
      .fetch<Providers>("/auth/providers")
      .then(p => setProviders(p))
      .catch(() =>
        setError("Something went wrong while fetching providers, try again.")
      )
      .finally(() => setLoading(false));
  }, []);

  return { error, loading, providers };
};

export const logout = () => (): void => {
  api.removeAuthToken();
  window.location.href = "/";
};

interface LoginOauthProps {
  setUser: (u: User) => void;
}

export interface LoginOauthReturnValue {
  user: User;
  accessToken: string;
  sessionToken: string;
}

// This one returns a function that returns another function.
// The first function is used to inject the api props. The second
// function produces an oauthlogin function based on the provider.
export const loginOauth = ({ setUser }: LoginOauthProps) => {
  return (provider: Provider): Promise<LoginOauthReturnValue> => {
    return new Promise((resolve, reject) => {
      let url = api.baseurl + `/auth/${provider}`;

      const title = "oauthWindow";
      let alreadyClosed = false;

      const onClose = (data: DataMessage) => {
        // This function gets called twice with different data for some reason.
        // This hack makes sure the second is disregarded.
        if (alreadyClosed) {
          return;
        }

        alreadyClosed = true;

        if (data?.sessionToken) {
          api.setAuthToken(data.sessionToken); // adds it to local storage
          setUser(data.user!);

          // Persist it for this session
          if (data.accessToken) {
            bitbucketApi.accessToken = data.accessToken!;
            gitlabApi.accessToken = data.accessToken;
            LocalStorage.set(LS_ACCESS_TOKEN, data.accessToken);
          }

          if (provider) {
            LocalStorage.set(LS_PROVIDER, provider);
          }

          resolve({
            user: data.user!,
            sessionToken: data.sessionToken,
            accessToken: data.accessToken!,
          });
        }

        if (!data?.success) {
          if (data.email === false) {
            reject(
              "We could not fetch your primary verified email from the provider. Make sure your email is verified."
            );
          } else if (data.error === "seats-full") {
            reject(
              "Your license does not allow more seats. Upgrade your plan to accept new users."
            );
          } else if (data.error === "account-too-new") {
            reject(
              "Your provider account is newly created. We do not accept new accounts. Please wait a few days."
            );
          } else if (data.error === "pending-approval-or-rejected") {
            reject(
              "Your account is pending approval or has been rejected by an administrator."
            );
          } else {
            reject("An error occurred while authenticating. Please retry.");
          }
        }
      };

      openPopup({ url, title, onClose });
    });
  };
};

interface FetchTeamsProps {
  user?: User;
  refreshToken: Number;
}

export const useFetchTeams = ({ user, refreshToken }: FetchTeamsProps) => {
  const [teams, setTeams] = useState<Team[]>();
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!user) {
      setLoading(false);
      return;
    }

    setLoading(true);
    setError("");

    api
      .fetch<Team[]>("/teams")
      .then(teams => {
        setTeams(teams);
      })
      .catch(res => {
        if (res.status === 402) {
          setTeams([
            {
              id: "personal",
              name: "No teams",
              isDefault: true,
              slug: "personal",
              currentUserRole: "owner",
            },
          ]);
        } else {
          setError("Something went wrong while fetching teams.");
        }
      })
      .finally(() => {
        setLoading(false);
      });
  }, [refreshToken, user]);

  return { teams, error, loading };
};

// Hook that uses the cache
export const useFetchInstanceDetails = (refreshToken?: number) => {
  const [details, setDetails] = useState<InstanceDetails>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();

  useEffect(() => {
    setError(undefined);

    api
      .fetch<InstanceDetails>("/instance")
      .then(d => {
        setDetails(d);
      })
      .catch(() => {
        setError("Something went wrong while fetching instance details.");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [refreshToken]);

  return { details, loading, error };
};
