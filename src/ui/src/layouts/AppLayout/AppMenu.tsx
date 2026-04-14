import { useMemo } from "react";
import { useLocation } from "react-router-dom";
import Box from "@mui/material/Box";
import ArrowBack from "@mui/icons-material/ArrowBack";
import AppName from "~/components/AppName";
import MenuLink from "~/components/MenuLink";
import { appMenuItems } from "./menu_items";

interface Props {
  app: App;
  team?: Team;
}

export default function AppMenu({ app, team }: Props) {
  const { pathname } = useLocation();

  const appMenu = useMemo(
    () => appMenuItems({ app, pathname }),
    [app, pathname],
  );

  return (
    <Box
      component="header"
      bgcolor="background.paper"
      sx={{
        display: "flex",
        flexDirection: "column",
        maxWidth: "100%",
        zIndex: 100,
        px: 2,
        py: 1,
      }}
    >
      <Box
        sx={{
          display: "flex",
          alignItems: "center",
        }}
      >
        <Box
          sx={{
            mr: 1,
            display: "flex",
            alignItems: "center",
          }}
        >
          <MenuLink
            sx={{
              mr: 1,
              px: { xs: 1, md: 1 },
              pr: { xs: 1, md: 2 },
              bgcolor: "container.transparent",
              color: "text.secondary",
            }}
            item={{
              text: team?.isDefault ? "My apps" : `${team?.name} Team Apps`,
              path: `/${team?.slug || "personal"}`,
              icon: <ArrowBack sx={{ fontSize: 18, mr: 1 }} />,
            }}
          />
          <MenuLink
            inline
            item={{
              text: <AppName imageSize={18} app={app} />,
              path: `/apps/${app.id}/environments/${app.defaultEnvId}`,
              isActive: pathname.includes(`/environments/${app.defaultEnvId}`),
            }}
          />
        </Box>
        <Box>
          {appMenu.map(item => (
            <MenuLink inline key={item.path} item={item} sx={{ mr: 1 }} />
          ))}
        </Box>
      </Box>
    </Box>
  );
}
