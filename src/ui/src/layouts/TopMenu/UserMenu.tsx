import { useContext } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Link from "@mui/material/Link";
import DarkModeIcon from "@mui/icons-material/DarkMode";
import LightModeIcon from "@mui/icons-material/LightMode";
import Button from "@mui/material/Button";
import ToggleButton from "@mui/material/ToggleButton";
import ToggleButtonGroup from "@mui/material/ToggleButtonGroup";
import { RootContext } from "~/pages/Root.context";

interface MenuItem {
  to: string;
  text: string;
  label?: string;
}

const menuItems: MenuItem[][] = [
  [
    { to: "/user/account", text: "Account" },
    {
      to: "https://www.stormkit.io/docs/welcome/getting-started",
      text: "Documentation",
    },
    { to: "/logout", text: "Log out" },
  ],
];

const footerItems: MenuItem[] = [
  { to: "https://twitter.com/stormkitio", text: "fab fa-twitter", label: "X" },
  {
    to: "https://www.youtube.com/channel/UC6C_-UuAiIWlGOIokT03lRQ",
    text: "fab fa-youtube",
    label: "YouTube",
  },
  {
    to: "https://discord.gg/6yQWhyY",
    text: "fab fa-discord",
    label: "Discord",
  },
  { to: "mailto:hello@stormkit.io", text: "fa fa-envelope", label: "Email" },
];

interface Props {
  user: User;
  onClick?: () => void;
}

export default function UserMenu({ user, onClick }: Props) {
  const { mode, setMode } = useContext(RootContext);

  return (
    <Box component="section" role="menu" sx={{ p: 2, minWidth: "250px" }}>
      <Box className="flex flex-col flex-1">
        <Box
          sx={{
            pb: 2,
            mb: 2,
            borderBottom: "1px solid",
            borderColor: "container.transparent",
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
          }}
        >
          <Box>
            <Typography>{user.fullName || user.displayName}</Typography>
            <Typography sx={{ color: "text.secondary" }}>
              {user.email}
            </Typography>
          </Box>
          {user.isAdmin && (
            <Button variant="contained" color="secondary" href="/admin">
              Admin
            </Button>
          )}
        </Box>
        <Box
          sx={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            borderBottom: "1px solid",
            borderColor: "container.transparent",
            mb: 2,
            pb: 2,
          }}
        >
          <Typography>Theme</Typography>
          <Box>
            <ToggleButtonGroup
              value={mode}
              exclusive
              sx={{ bgcolor: "container.paper" }}
              onChange={(_, val) => {
                if (val !== null) {
                  setMode(val);
                }
              }}
              aria-label="display mode"
            >
              <ToggleButton
                value="dark"
                aria-label="24 hours"
                size="small"
                sx={{
                  color: "text.primary",
                }}
              >
                <DarkModeIcon sx={{ fontSize: 14 }} />
              </ToggleButton>
              <ToggleButton
                value="light"
                aria-label="Light"
                size="small"
                sx={{
                  color: "text.primary",
                }}
              >
                <LightModeIcon sx={{ fontSize: 14 }} />
              </ToggleButton>
            </ToggleButtonGroup>
          </Box>
        </Box>
        {menuItems.map((section, index) => (
          <Box
            key={index}
            sx={{
              borderBottom: "1px solid",
              borderColor: "container.transparent",
              mb: 2,
            }}
          >
            {section.map(item => (
              <Box key={item.text} sx={{ pb: 2 }}>
                <Link
                  href={item.to}
                  sx={{ display: "block" }}
                  onClick={onClick}
                >
                  {item.text}
                </Link>
              </Box>
            ))}
          </Box>
        ))}
      </Box>
      <Box sx={{ textAlign: "center" }}>
        {footerItems.map(item => (
          <Link
            key={item.to}
            href={item.to}
            target="_blank"
            rel="noreferrer nooopener"
            aria-label={item.label}
            sx={{
              mr: 2,
              fontSize: 20,
              display: "inline-block",
              ":last-child": { mr: 0 },
            }}
          >
            <Box component="span" className={item.text} />
          </Link>
        ))}
      </Box>
    </Box>
  );
}
