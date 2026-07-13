import { useContext, useEffect, useState } from "react";
import Box from "@mui/material/Box";
import Link from "@mui/material/Link";
import MenuItem from "@mui/material/MenuItem";
import Select from "@mui/material/Select";
import Typography from "@mui/material/Typography";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import ClickAwayListener from "@mui/material/ClickAwayListener";
import Check from "@mui/icons-material/Check";
import KeyboardArrowDown from "@mui/icons-material/KeyboardArrowDown";
import KeyboardArrowUp from "@mui/icons-material/KeyboardArrowUp";
import { AuthContext } from "~/pages/auth/Auth.context";
import AppName from "~/components/AppName";
import { useFetchAppList } from "~/pages/apps/actions";
import { recentApps, recordRecentApp } from "./recent_apps";

interface Props {
  app: App;
  team?: Team;
}

interface MenuProps {
  app: App;
  teamId?: string;
  onClickAway: () => void;
}

const sectionHeader = {
  height: "50px",
  borderBottom: "1px solid",
  borderColor: "container.transparent",
  display: "flex",
  alignItems: "center",
};

const appLink = {
  p: 1,
  flex: 1,
  minWidth: "220px",
  maxWidth: "calc(100vw - 32px)",
  borderRadius: 1,
  display: "flex",
  alignItems: "center",
  ":hover": {
    bgcolor: "container.transparent",
    cursor: "pointer",
    color: "inherit",
  },
};

function AppSelectorMenu({ app, teamId, onClickAway }: MenuProps) {
  const { teams } = useContext(AuthContext);
  const [selectedTeamId, setSelectedTeamId] = useState(teamId);
  const { apps } = useFetchAppList({ teamId: selectedTeamId });
  const recent = recentApps().filter(a => a.id !== app.id);

  return (
    <ClickAwayListener onClickAway={onClickAway}>
      <Box>
        <Box sx={sectionHeader}>
          {/* disablePortal: a portal-based menu would trigger the
              ClickAwayListener and close the whole dropdown. */}
          <Select
            fullWidth
            size="small"
            variant="outlined"
            value={selectedTeamId || ""}
            inputProps={{ "aria-label": "Select team" }}
            MenuProps={{ disablePortal: true }}
            onChange={e => setSelectedTeamId(e.target.value as string)}
          >
            {teams?.map(t => (
              <MenuItem key={t.id} value={t.id}>
                {t.isDefault ? "Personal" : t.name}
              </MenuItem>
            ))}
          </Select>
        </Box>
        <Box role="list">
          {apps.map(a => (
            <Box
              key={a.id}
              data-testid={`app-${a.id}`}
              sx={{ display: "flex", alignItems: "center", my: 2 }}
            >
              <Check
                sx={{
                  fontSize: 14,
                  mr: 1,
                  cursor: "default",
                  visibility: a.id === app.id ? "visible" : "hidden",
                }}
              />
              <Link href={`/apps/${a.id}`} onClick={onClickAway} sx={appLink}>
                <AppName app={a} imageSize={18} />
              </Link>
            </Box>
          ))}
        </Box>
        <Box sx={sectionHeader}>
          <Typography variant="h2" sx={{ flex: 1, pl: 1 }} color="text.primary">
            Recent apps
          </Typography>
        </Box>
        {recent.length > 0 ? (
          <Box role="list">
            {recent.map(a => (
              <Box
                key={a.id}
                data-testid={`recent-app-${a.id}`}
                sx={{ display: "flex", alignItems: "center", my: 2 }}
              >
                <Box sx={{ width: 14, mr: 1 }} />
                <Link href={`/apps/${a.id}`} onClick={onClickAway} sx={appLink}>
                  <AppName app={a} imageSize={18} />
                </Link>
              </Box>
            ))}
          </Box>
        ) : (
          <Typography sx={{ p: 1, my: 1 }} color="text.secondary">
            No recently visited apps.
          </Typography>
        )}
      </Box>
    </ClickAwayListener>
  );
}

export default function AppSelector({ app, team }: Props) {
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const Arrow = isMenuOpen ? KeyboardArrowUp : KeyboardArrowDown;

  useEffect(() => {
    recordRecentApp(app);
  }, [app.id]);

  return (
    <Tooltip
      arrow
      open={isMenuOpen}
      placement="bottom-start"
      slotProps={{
        popper: {
          modifiers: [
            { name: "offset", options: { offset: [0, 12] } },
            // Position with top/left instead of a CSS transform: the
            // select menu inside is position fixed, and a transformed
            // ancestor would offset it to the wrong place.
            { name: "computeStyles", options: { gpuAcceleration: false } },
          ],
        },
        tooltip: { sx: { maxWidth: "none" } },
      }}
      title={
        <Box sx={{ p: 1 }}>
          <AppSelectorMenu
            app={app}
            teamId={team?.id}
            onClickAway={() => {
              setIsMenuOpen(false);
            }}
          />
        </Box>
      }
    >
      <IconButton
        aria-label="Select app"
        type="button"
        onClick={e => {
          e.preventDefault();
          setIsMenuOpen(!isMenuOpen);
        }}
      >
        <Arrow sx={{ fontSize: 16 }} />
      </IconButton>
    </Tooltip>
  );
}
