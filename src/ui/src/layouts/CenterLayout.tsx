import React, { useContext } from "react";
import Box from "@mui/material/Box";
import { AuthContext } from "~/pages/auth/Auth.context";
import { useSelectedTeam } from "./TopMenu/Teams/actions";
import TopMenu from "./TopMenu";

interface Props {
  children: React.ReactNode;
}

export default function CenterLayout({ children }: Props) {
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
          <TopMenu team={selectedTeam} />
        </Box>
      )}
      <Box sx={{ flex: 1, px: { xs: 2, md: 0 } }}>{children}</Box>
    </Box>
  );
}
