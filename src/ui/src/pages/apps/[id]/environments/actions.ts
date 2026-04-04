import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

interface FetchEnvironmentsProps {
  app?: App;
  refreshToken?: number;
}

interface FetchEnvironmentsReturnValue {
  environments: Array<Environment>;
  loading: boolean;
  error: string | null;
}

interface FetchEnvironmentsAPIResponse {
  environments: Array<Environment>;
}

export const useFetchEnvironments = ({
  app,
  refreshToken,
}: FetchEnvironmentsProps): FetchEnvironmentsReturnValue => {
  const [environments, setEnvironments] = useState<Array<Environment>>([]);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    let unmounted = false;

    if (!app?.id) {
      return;
    }

    setLoading(true);
    setError(null);

    api
      .fetch<FetchEnvironmentsAPIResponse>(`/v1/envs?appId=${app.id}`)
      .then(res => {
        if (unmounted !== true) {
          setEnvironments(
            res.environments.map(e => ({
              ...e,
              name: e.env,
            })),
          );
        }
      })
      .finally(() => {
        if (unmounted !== true) {
          setLoading(false);
        }
      });

    return () => {
      unmounted = true;
    };
  }, [app?.id, app?.displayName, refreshToken]);

  return { environments, error, loading };
};

interface FetchStatusProps {
  app: App;
  environment: Environment;
  domain: string;
}

interface FetchStatusReturnValue {
  status?: number;
  loading: boolean;
}

interface FetchStatusAPIResponse {
  status: number;
}

export const isEmpty = (val?: boolean | Array<unknown>): boolean => {
  if (typeof val === "boolean") {
    return !val;
  }

  if (Array.isArray(val)) {
    return val.length === 0;
  }

  return !val;
};

export const useFetchStatus = ({
  app,
  environment,
  domain,
}: FetchStatusProps): FetchStatusReturnValue => {
  const [status, setStatus] = useState<number>();
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    let unmounted = false;

    if (isEmpty(environment.published)) {
      return;
    }

    setLoading(true);

    api
      .post<FetchStatusAPIResponse>("/app/proxy", {
        url: domain,
        appId: app.id,
      })
      .then(res => {
        if (!unmounted) {
          setStatus(res.status);
        }
      })
      .catch(() => {
        // do nothing
      })
      .finally(() => {
        if (!unmounted) {
          setLoading(false);
        }
      });

    return () => {
      unmounted = true;
    };
  }, [domain, app.id, environment.published]);

  return { status, loading };
};
