import React, { useContext } from "react";
import { Route, Routes } from "react-router-dom";
import Box from "@mui/material/Box";
import AppContextProvider, { AppContext } from "~/pages/apps/[id]/App.context";
import { AuthContext } from "~/pages/auth/Auth.context";
import TopMenu from "../TopMenu";
import { useSelectedTeam } from "../TopMenu/Teams/actions";
import AppMenu from "./AppMenu";
import EnvMenu from "./EnvMenu";
import { routes } from "./routes";

export function AppLayout() {
  const { app, environments } = useContext(AppContext);
  const { teams } = useContext(AuthContext);
  const selectedTeam = useSelectedTeam({ teams, app });

  if (environments.length === 0) {
    return <></>;
  }

  return (
    <Box
      sx={{
        display: "flex",
        flexDirection: "column",
        minHeight: "100vh",
        width: "100%",
      }}
    >
      <TopMenu
        app={app}
        team={selectedTeam}
        submenu={<AppMenu team={selectedTeam} app={app} />}
      />
      <Box
        sx={{
          display: "flex",
          flexDirection: { xs: "column", md: "row" },
          width: "100%",
          flex: 1,
        }}
      >
        <EnvMenu />
        <Box sx={{ px: 2, width: "100%" }}>
          <Box
            component="section"
            sx={{
              flex: 1,
              my: 2,
              mx: "auto",
              width: "100%",
              display: "flex",
              flexDirection: "column",
              alignItems: "flex-start",
              maxWidth: "1024px",
            }}
          >
            <Box sx={{ width: "100%" }}>
              <Routes>
                {routes.map(r => (
                  <Route {...r} key={r.path} />
                ))}
              </Routes>
            </Box>
          </Box>
        </Box>
      </Box>
    </Box>
  );
}

const WithAppProvider: React.FC = () => (
  <AppContextProvider>
    <AppLayout />
  </AppContextProvider>
);

export default WithAppProvider;
