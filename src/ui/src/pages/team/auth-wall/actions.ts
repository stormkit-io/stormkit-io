import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

export type AuthWallMode = "all" | "dev" | "";

export interface TeamAuthWallLogin {
  id: string;
  email: string;
  metadata: string;
  magicLink?: boolean;
  hasPassword?: boolean;
  allApps: boolean;
  appIds: string[];
  lastLogin?: number;
  createdAt?: number;
}

export interface TeamAuthWallEnv {
  id: string;
  name: string;
  authWall: AuthWallMode;
}

export interface TeamAuthWallApp {
  id: string;
  displayName: string;
  environments: TeamAuthWallEnv[];
}

const isPaymentRequired = (res: unknown) =>
  (res as Response | undefined)?.status === 402;

interface UseFetchTeamEnvsProps {
  teamId?: string;
  refreshToken?: number;
}

export const useFetchTeamEnvs = ({
  teamId,
  refreshToken,
}: UseFetchTeamEnvsProps) => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const [paymentRequired, setPaymentRequired] = useState(false);
  const [apps, setApps] = useState<TeamAuthWallApp[]>([]);

  useEffect(() => {
    if (!teamId) {
      return;
    }

    // Responses for a previous team are dropped.
    let cancelled = false;

    setLoading(true);
    setError(undefined);

    api
      .fetch<{ apps: TeamAuthWallApp[] }>(
        `/team/auth-wall/envs?teamId=${teamId}`,
      )
      .then(({ apps }) => {
        if (!cancelled) {
          setApps(apps);
        }
      })
      .catch(res => {
        if (cancelled) {
          return;
        }

        if (isPaymentRequired(res)) {
          setPaymentRequired(true);
          return;
        }

        setError("Something went wrong while fetching the environments.");
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [teamId, refreshToken]);

  return { loading, error, paymentRequired, apps };
};

interface UseFetchTeamLoginsProps {
  teamId?: string;
  refreshToken?: number;
}

export const useFetchTeamLogins = ({
  teamId,
  refreshToken,
}: UseFetchTeamLoginsProps) => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const [logins, setLogins] = useState<TeamAuthWallLogin[]>([]);
  const [hasNextPage, setHasNextPage] = useState(false);
  const [from, setFrom] = useState(0);

  useEffect(() => {
    setFrom(0);
  }, [teamId, refreshToken]);

  useEffect(() => {
    if (!teamId) {
      return;
    }

    // Responses for a previous team or page are dropped.
    let cancelled = false;

    setLoading(true);
    setError(undefined);

    api
      .fetch<{ logins: TeamAuthWallLogin[]; hasNextPage: boolean }>(
        `/team/auth-wall/logins?teamId=${teamId}&from=${from}`,
      )
      .then(data => {
        if (cancelled) {
          return;
        }

        setLogins(prev =>
          from === 0 ? data.logins : [...prev, ...data.logins],
        );
        setHasNextPage(data.hasNextPage);
      })
      .catch(() => {
        if (!cancelled) {
          setError("Something went wrong while fetching the logins.");
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [teamId, refreshToken, from]);

  return {
    loading,
    error,
    logins,
    hasNextPage,
    loadMore: () => setFrom(logins.length),
  };
};

interface CreateTeamLoginProps {
  teamId: string;
  email: string;
  password: string;
  metadata: string;
  magicLink: boolean;
  allApps: boolean;
  appIds: string[];
}

export const createTeamLogin = (props: CreateTeamLoginProps) => {
  return api.post<{ login: TeamAuthWallLogin }>(
    "/team/auth-wall/logins",
    props,
  );
};

interface UpdateTeamLoginProps {
  teamId: string;
  id: string;
  password?: string;
  removePassword?: boolean;
  metadata?: string;
  magicLink?: boolean;
  allApps?: boolean;
  appIds?: string[];
}

export const updateTeamLogin = (props: UpdateTeamLoginProps) => {
  return api.patch<{ login: TeamAuthWallLogin }>(
    "/team/auth-wall/logins",
    props,
  );
};

interface DeleteTeamLoginsProps {
  teamId: string;
  loginIds: string[];
}

export const deleteTeamLogins = ({
  teamId,
  loginIds,
}: DeleteTeamLoginsProps) => {
  return api.delete(
    `/team/auth-wall/logins?teamId=${teamId}&id=${loginIds.join(",")}`,
  );
};

interface UpdateEnvAuthWallProps {
  teamId: string;
  envId: string;
  authWall: AuthWallMode;
}

export const updateEnvAuthWall = (props: UpdateEnvAuthWallProps) => {
  return api.put("/team/auth-wall/envs", props);
};
