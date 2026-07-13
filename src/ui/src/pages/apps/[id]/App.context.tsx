import React, { createContext } from "react";
import { useParams } from "react-router-dom";
import Link from "@mui/material/Link";
import Typography from "@mui/material/Typography";
import Spinner from "~/components/Spinner";
import Error404 from "~/components/Errors/Error404";
import { useFetchApp } from "./actions";
import { useFetchEnvironments } from "./environments/actions";

export interface AppContextProps {
  app: App;
  environments: Array<Environment>;
  setRefreshToken: (val: number) => void;
}

export const AppContext = createContext<AppContextProps>({
  app: {} as App,
  environments: [],
  setRefreshToken: () => {},
});

interface Props {
  children: React.ReactNode;
}

export default function AppProvider({ children }: Props) {
  const appId = useParams().id;
  const { app, error, loading, refreshToken, setRefreshToken } = useFetchApp({
    appId,
  });

  const envs = useFetchEnvironments({ app, refreshToken });

  if (loading) {
    return <Spinner primary pageCenter />;
  }

  if (error || envs.error) {
    return (
      <Typography color="error" sx={{ textAlign: "center", mt: 2 }}>
        Something went wrong on our side. Please try again. If the problem
        persists reach us out through Discord or email.
      </Typography>
    );
  }

  if (!app) {
    return (
      <Error404>
        <Typography sx={{ fontSize: "inherit" }}>
          We couldn't find this app <br />
          Click to go back to{" "}
          <Link href="/" sx={{ fontSize: "inherit" }}>
            My Apps
          </Link>
        </Typography>
      </Error404>
    );
  }

  return (
    <AppContext.Provider
      value={{
        app,
        environments: envs.environments,
        setRefreshToken,
      }}
    >
      {children}
    </AppContext.Provider>
  );
}
