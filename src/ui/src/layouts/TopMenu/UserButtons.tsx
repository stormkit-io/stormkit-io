import React, { useContext, useEffect, useState } from "react";
import DOMPurify from "dompurify";
import { marked } from "marked";
import { AuthContext } from "~/pages/auth/Auth.context";
import api from "~/utils/api/Api";
import { Notifications } from "@mui/icons-material";
import { useTheme } from "@mui/material/styles";
import Box from "@mui/material/Box";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import ClickAwayListener from "@mui/material/ClickAwayListener";
import UserAvatar from "~/components/UserAvatar";
import SideBar from "~/components/SideBar";
import Spinner from "~/components/Spinner";
import UserMenu from "./UserMenu";

const UserButtons: React.FC = () => {
  const theme = useTheme();
  const { user } = useContext(AuthContext);
  const [isNewsOpen, toggleNews] = useState(false);
  const [isUserMenuOpen, toggleUserMenu] = useState(false);
  const [newsHtml, setNewsHtml] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if (!isNewsOpen || newsHtml) {
      return;
    }

    setIsLoading(true);

    api
      .fetch<{ markdown: string }>("/changelog")
      .then(data => {
        const raw = (marked.parse(data.markdown || "") as string)
          .replace(/src="(\/[^"]+)"/g, 'src="https://www.stormkit.io$1"')
          .replace(/href="(\/[^"]+)"/g, 'href="https://www.stormkit.io$1"');

        const html = DOMPurify.sanitize(raw, {
          ALLOWED_TAGS: [
            "h1",
            "h2",
            "h3",
            "h4",
            "h5",
            "h6",
            "p",
            "a",
            "ul",
            "ol",
            "li",
            "strong",
            "em",
            "b",
            "i",
            "code",
            "pre",
            "blockquote",
            "img",
            "br",
            "hr",
          ],
          ALLOWED_ATTR: ["href", "src", "alt", "title"],
          ALLOW_DATA_ATTR: false,
        });

        setNewsHtml(html);
      })
      .catch(() => {
        setNewsHtml("<p>Failed to load changelog.</p>");
      })
      .finally(() => {
        setIsLoading(false);
      });
  }, [isNewsOpen, newsHtml]);

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
          {newsHtml && (
            <Typography
              component="div"
              dangerouslySetInnerHTML={{ __html: newsHtml }}
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
            />
          )}
        </Box>
      </SideBar>
    </>
  );
};

export default UserButtons;
