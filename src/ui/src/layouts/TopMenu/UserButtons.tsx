import React, { useContext, useEffect, useState } from "react";
import { AuthContext } from "~/pages/auth/Auth.context";
import api from "~/utils/api/Api";
import { Notifications } from "@mui/icons-material";
import { useTheme } from "@mui/material/styles";
import Box from "@mui/material/Box";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import ClickAwayListener from "@mui/material/ClickAwayListener";
import UserAvatar from "~/components/UserAvatar";
import Markdown from "~/components/Markdown";
import SideBar from "~/components/SideBar";
import Spinner from "~/components/Spinner";
import UserMenu from "./UserMenu";

// The changelog is authored relative to www.stormkit.io, so its links and
// images are absolutized before rendering. Applied to the markdown source,
// which covers both markdown syntax and any raw HTML embedded in it.
function absolutize(markdown: string): string {
  return markdown
    .replace(/\]\(\//g, "](https://www.stormkit.io/")
    .replace(/src="(\/[^"]+)"/g, 'src="https://www.stormkit.io$1"')
    .replace(/href="(\/[^"]+)"/g, 'href="https://www.stormkit.io$1"');
}

const UserButtons: React.FC = () => {
  const theme = useTheme();
  const { user } = useContext(AuthContext);
  const [isNewsOpen, toggleNews] = useState(false);
  const [isUserMenuOpen, toggleUserMenu] = useState(false);
  const [news, setNews] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if (!isNewsOpen || news) {
      return;
    }

    setIsLoading(true);

    api
      .fetch<{ markdown: string }>("/changelog")
      .then(data => {
        setNews(absolutize(data.markdown || ""));
      })
      .catch(() => {
        setNews("Failed to load changelog.");
      })
      .finally(() => {
        setIsLoading(false);
      });
  }, [isNewsOpen, news]);

  if (!user) {
    return <></>;
  }

  return (
    <>
      <ClickAwayListener
        onClickAway={() => {
          toggleNews(false);
        }}
      >
        <Tooltip title="What's new?" placement="bottom" arrow>
          <IconButton
            onClick={() => {
              toggleUserMenu(false);
              toggleNews(!isNewsOpen);
            }}
          >
            <Notifications />
          </IconButton>
        </Tooltip>
      </ClickAwayListener>

      <Tooltip
        title={
          <ClickAwayListener
            onClickAway={() => {
              toggleUserMenu(false);
            }}
          >
            <div>
              <UserMenu user={user} onClick={() => toggleUserMenu(false)} />
            </div>
          </ClickAwayListener>
        }
        placement="bottom-end"
        open={isUserMenuOpen}
        arrow
      >
        <IconButton
          onClick={() => {
            toggleUserMenu(!isUserMenuOpen);
            toggleNews(false);
          }}
        >
          <UserAvatar user={user} />
        </IconButton>
      </Tooltip>

      <SideBar isOpen={isNewsOpen}>
        <Box
          sx={{ position: "relative", height: "100%", overflow: "auto", p: 2 }}
        >
          {isLoading && (
            <Box
              sx={{
                position: "absolute",
                top: "50%",
                left: "50%",
                transform: "translate(-50%, -50%)",
              }}
            >
              <Spinner />
            </Box>
          )}
          {news && (
            <Markdown
              allowImages
              sx={{
                "& h2": { mt: 2, mb: 1, fontSize: "1.1rem", fontWeight: 600 },
                "& p": { mb: 1.5, lineHeight: 1.6 },
                "& a": { color: theme.palette.primary.main },
                "& code": {
                  bgcolor: theme.palette.action.selected,
                  px: 0.5,
                  borderRadius: 0.5,
                  fontSize: "0.875rem",
                },
                "& img": { maxWidth: "100%", height: "auto", display: "block" },
              }}
            >
              {news}
            </Markdown>
          )}
        </Box>
      </SideBar>
    </>
  );
};

export default UserButtons;
