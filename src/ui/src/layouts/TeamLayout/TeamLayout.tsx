import React, { useContext } from "react";
import Box from "@mui/material/Box";
import { AuthContext } from "~/pages/auth/Auth.context";
import { useSelectedTeam } from "../TopMenu/Teams/actions";
import TeamMenu from "./TeamNav";
import TopMenu from "../TopMenu";
import Error404 from "~/components/Errors/Error404";

interface Props {
  children: React.ReactNode;
}

export default function TeamLayout({ children }: Props) {
  const { user, teams } = useContext(AuthContext);
  const selectedTeam = useSelectedTeam({ teams });

  return (
    <Box
      component="main"
      sx={{
        display: "flex",
        flexDirection: "column",
        minHeight: "100vh",
        width: "100%",
      }}
    >
      {user && (
        <Box
          bgcolor="background.paper"
          sx={{
            display: "flex",
            position: "sticky",
            top: 0,
            zIndex: 50,
            boxShadow: 2,
          }}
        >
          <TopMenu team={selectedTeam} submenu={<TeamMenu />} />
        </Box>
      )}
      <Box
        sx={{
          mt: 2,
          flex: 1,
          px: { xs: 2, md: 0 },
          display: "flex",
          alignItems: user ? "flex-start" : "center",
          justifyContent: "center",
        }}
      >
        {selectedTeam ? children : <Error404 />}
      </Box>
    </Box>
  );
}
