import React from "react";
import Box from "@mui/material/Box";
import Link from "@mui/material/Link";
import Logo from "~/components/Logo";
import UserButtons from "./UserButtons";
import TeamToggle from "./Teams/TeamToggle";
import { grey } from "@mui/material/colors";

interface Props {
  children?: React.ReactNode;
  submenu?: React.ReactNode;
  team?: Team;
  app?: App;
}

export default function TopMenu({ children, submenu, app, team }: Props) {
  const slug = `/${team?.slug || ""}`;

  return (
    <Box
      sx={{
        width: "100%",
        borderBottom: `1px solid ${grey[900]}`,
      }}
    >
      <Box
        id="top-bar"
        bgcolor="background.paper"
        component="nav"
        sx={{
          display: "flex",
          alignItems: "center",
          boxShadow: 2,
          m: "auto",
          px: 2,
          py: 1,
          width: "100%",
        }}
      >
        <Box sx={{ display: "flex", alignItems: "center" }}>
          <Link href={slug} sx={{ flex: "1", mr: 1 }}>
            <Logo iconSize={28} iconOnly />
          </Link>
          <TeamToggle app={app} />
        </Box>
        <Box sx={{ flex: 1 }}>{children}</Box>
        <Box sx={{ display: "flex", alignItems: "center" }}>
          <UserButtons />
        </Box>
      </Box>
      {submenu}
    </Box>
  );
}
