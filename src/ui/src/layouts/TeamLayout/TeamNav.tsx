import { useMemo, useContext } from "react";
import { useLocation } from "react-router-dom";
import Box from "@mui/material/Box";
import { AuthContext } from "~/pages/auth/Auth.context";
import MenuLink from "~/components/MenuLink";
import { useSelectedTeam } from "../TopMenu/Teams/actions";

interface Path {
  path: string;
  icon?: React.ReactNode;
  text: React.ReactNode;
  isActive?: boolean;
}

const teamMenuItems = ({
  team,
  pathname,
}: {
  team: Team;
  pathname: string;
}): Path[] => {
  const { slug } = team;

  const items: Path[] = [
    {
      path: `/${team.slug}`,
      text: team.isDefault ? "My apps" : `${team.name} Team Apps`,
      isActive: ["/", `/${slug}`].indexOf(pathname) > -1,
    },
  ];

  items.push({
    path: `/${slug}/deployments`,
    text: "All Deployments",
    isActive: pathname.includes("/deployments"),
  });

  items.push({
    path: `/${slug}/feed`,
    text: "Activity Feed",
    isActive: pathname.includes("/feed"),
  });

  items.push({
    path: `/${slug}/insights`,
    text: "Insights",
    isActive: pathname.includes("/insights"),
  });

  if (!team.isDefault) {
    items.push({
      path: `/${slug}/settings`,
      text: "Settings",
      isActive: pathname.includes("/settings"),
    });
  }

  return items;
};

export default function TeamMenu() {
  const { pathname } = useLocation();
  const { teams } = useContext(AuthContext);
  const selectedTeam = useSelectedTeam({ teams });

  const teamMenu = useMemo(() => {
    if (selectedTeam) {
      return teamMenuItems({ team: selectedTeam, pathname });
    }

    return [];
  }, [selectedTeam, pathname]);

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
        {teamMenu.map(item => (
          <MenuLink key={item.path} item={item} sx={{ mr: 1 }} />
        ))}
      </Box>
    </Box>
  );
}
