import type { LoginOauthReturnValue } from "./actions";
import CircularProgress from "@mui/material/CircularProgress";
import React, { createContext, useEffect, useState } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import * as actions from "./actions";
import type { Providers } from "./actions";
import UpdateSnackbar from "./UpdateSnackbar";

const {
  loginOauth,
  useFetchUser,
  useFetchTeams,
  logout,
  useFetchActiveProviders,
} = actions;

export interface AuthContextProps {
  user?: User;
  authError?: string | null;
  accounts?: Array<ConnectedAccount>;
  metrics?: UserMetrics;
  providers?: Providers;
  logout?: () => void;
  loginOauth?: (p: Provider) => Promise<LoginOauthReturnValue>;
  reloadTeams?: () => void;
  teams?: Team[];
}

export const AuthContext = createContext<AuthContextProps>({});

interface Props {
  children: React.ReactNode;
}

export default function ContextProvider({ children }: Props) {
  const navigate = useNavigate();
  const [refreshToken, setTeamsRefreshToken] = useState(0);
  const { pathname, search } = useLocation();
  const { error, loading, user, metrics, accounts, ...fns } = useFetchUser();
  const {
    providers,
    loading: pLoading,
    error: pError,
  } = useFetchActiveProviders();
  const { teams, loading: tLoading } = useFetchTeams({ refreshToken, user });

  const shouldRedirect = !loading && !user && pathname !== "/auth";

  useEffect(() => {
    if (shouldRedirect) {
      const encoded =
        pathname !== "/" && pathname !== "/logout"
          ? encodeURIComponent(`${pathname}${search}`)
          : undefined;

      navigate(`/auth${encoded ? `?redirect=${encoded}` : ""}`);
    }
  }, [shouldRedirect]);

  if (loading || tLoading || pLoading) {
    return (
      <CircularProgress
        sx={{
          position: "absolute",
          top: "50%",
          left: "50%",
          transform: "translate(-50%, -50%)",
          m: 0,
        }}
      />
    );
  }

  // Redirect the user to the console login if he/she
  // is not logged in and the pathname is not auth.
  if (shouldRedirect) {
    return null;
  }

  return (
    <AuthContext.Provider
      value={{
        user,
        accounts,
        authError: error || pError,
        teams,
        providers,
        metrics,
        reloadTeams: () => setTeamsRefreshToken(Date.now()),
        logout: logout(), // This function can be removed, it's no longer being injected something.
        loginOauth: loginOauth({ ...fns }),
      }}
    >
      {children}
      {user && <UpdateSnackbar />}
    </AuthContext.Provider>
  );
}
