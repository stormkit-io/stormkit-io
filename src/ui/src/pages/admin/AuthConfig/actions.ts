import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

export type SignUpMode = "on" | "off" | "waitlist";

export interface AuthConfig {
  whitelist: string[];
  signUpMode: SignUpMode;
}

export const useFetchAuthConfig = () => {
  const [config, setConfig] = useState<AuthConfig>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();

  useEffect(() => {
    api
      .fetch<AuthConfig>("/admin/users/sign-up-mode")
      .then(c => {
        setConfig(c);
      })
      .catch(() => {
        setError(
          "Something went wrong while fetching user management configuration. Please try again later."
        );
      })
      .finally(() => {
        setLoading(false);
      });
  }, []);

  return { config, loading, error };
};

interface UpdateAuthConfigProps {
  whitelist: string[];
  signUpMode: SignUpMode;
}

export const updateAuthConfig = ({
  whitelist,
  signUpMode,
}: UpdateAuthConfigProps) => {
  return api.post("/admin/users/sign-up-mode", { whitelist, signUpMode });
};

interface FetchPendingUsersProps {
  refreshToken?: number;
}

export const useFetchPendingUsers = ({
  refreshToken,
}: FetchPendingUsersProps) => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const [users, setUsers] = useState<User[]>([]);

  useEffect(() => {
    api
      .fetch<{ users: User[] }>("/admin/users/pending")
      .then(({ users }) => {
        setUsers(users);
      })
      .catch(() => {
        setError("Something went wrong while fetching pending users.");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [refreshToken]);

  return { loading, error, users };
};
