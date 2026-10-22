import { useContext, useEffect, useState } from "react";
import Box from "@mui/material/Box";
import Link from "@mui/material/Link";
import LockIcon from "@mui/icons-material/Lock";
import LockOpenIcon from "@mui/icons-material/LockOpen";
import { AuthContext } from "~/pages/auth/Auth.context";
import { RootContext } from "~/pages/Root.context";
import { useSelectedTeam } from "~/layouts/TopMenu/Teams/actions";
import api from "~/utils/api/Api";

interface Props {
  app: App;
  environment: Environment;
}

const labels: Record<string, string> = {
  "": "Off",
  dev: "Previews",
  all: "All URLs",
};

// AuthWallBadge shows the mode of the environment's Auth Wall and links to the
// team page where it is managed.
export default function AuthWallBadge({ app, environment }: Props) {
  const { teams } = useContext(AuthContext);
  const { details } = useContext(RootContext);
  const team = useSelectedTeam({ teams, app });
  const [mode, setMode] = useState<string>();
  const isCommunity = details?.license?.edition === "community";

  useEffect(() => {
    if (isCommunity) {
      return;
    }

    // Responses for a previous environment are dropped.
    let cancelled = false;

    setMode(undefined);

    api
      .fetch<{ authwall: string }>(
        `/auth-wall/config?appId=${app.id}&envId=${environment.id}`,
      )
      .then(({ authwall }) => {
        if (!cancelled) {
          setMode(authwall || "");
        }
      })
      .catch(() => {});

    return () => {
      cancelled = true;
    };
  }, [app.id, environment.id, isCommunity]);

  if (isCommunity || mode === undefined || !team) {
    return null;
  }

  const Icon = mode ? LockIcon : LockOpenIcon;

  return (
    <Box
      sx={{
        pr: 2,
        mr: 2,
        borderRight: "1px solid",
        borderColor: "container.transparent",
        display: { xs: "none", md: "block" },
      }}
    >
      <Link
        href={`/${team.slug}/auth-wall?appId=${app.id}`}
        color="inherit"
        sx={{ display: "inline-flex", alignItems: "center", gap: 0.5 }}
      >
        <Icon sx={{ fontSize: 14 }} />
        Auth Wall: {labels[mode] ?? mode}
      </Link>
    </Box>
  );
}
