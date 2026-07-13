import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

interface UseFetchAuthWallConfig {
  appId: string;
  envId: string;
  refreshToken?: number;
}

export type AuthWallConfig = "dev" | "all" | "";

export const useFetchAuthWallConfig = ({
  appId,
  envId,
  refreshToken,
}: UseFetchAuthWallConfig) => {
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string>();
  const [config, setConfig] = useState<AuthWallConfig>();

  useEffect(() => {
    api
      .fetch<{ authwall: AuthWallConfig }>(
        `/auth-wall/config?appId=${appId}&envId=${envId}`
      )
      .then(({ authwall }) => {
        setConfig(authwall);
      })
      .catch(() => {
        setError("Something went wrong while fetching the auth wall config");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [appId, envId, refreshToken]);

  return { loading, error, config };
};

interface UpdateAuthWallConfigProps {
  appId: string;
  envId: string;
  authwall: AuthWallConfig;
}

export const updateAuthWallConfig = ({
  appId,
  envId,
  authwall,
}: UpdateAuthWallConfigProps) => {
  return api.post("/auth-wall/config", {
    appId,
    envId,
    authwall,
  });
};

interface CreateNewLoginProps {
  appId: string;
  envId: string;
  email: string;
  password: string;
}

export const createNewLogin = ({
  appId,
  envId,
  email,
  password,
}: CreateNewLoginProps) => {
  return api.post("/auth-wall", {
    appId,
    envId,
    email,
    password,
  });
};

interface UseFetchAuthWallLogins {
  appId: string;
  envId: string;
  refreshToken?: number;
}

interface UserLogin {
  id: string;
  email: string;
  lastLogin?: number;
}

export const useFetchLogins = ({
  appId,
  envId,
  refreshToken,
}: UseFetchAuthWallLogins) => {
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string>();
  const [logins, setLogins] = useState<UserLogin[]>([]);

  useEffect(() => {
    setLoading(true);

    api
      .fetch<{ logins: UserLogin[] }>(
        `/auth-wall?appId=${appId}&envId=${envId}`
      )
      .then(({ logins }) => {
        setLogins(logins);
      })
      .catch(() => {
        setError("Something went wrong while fetching the auth wall logins");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [appId, envId, refreshToken]);

  return { loading, error, logins };
};

interface DeleteLoginsProps {
  appId: string;
  envId: string;
  loginIds: string[];
}

export const deleteLogins = ({ appId, envId, loginIds }: DeleteLoginsProps) => {
  return api.delete(
    `/auth-wall?appId=${appId}&envId=${envId}&id=${loginIds.join(",")}`
  );
};
