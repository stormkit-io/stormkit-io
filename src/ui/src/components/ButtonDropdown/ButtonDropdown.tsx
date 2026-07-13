import React, { useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import ArrowDropDown from "@mui/icons-material/ArrowDropDown";
import ArrowDropUp from "@mui/icons-material/ArrowDropUp";
import Link from "@mui/material/Link";
import Tooltip from "@mui/material/Tooltip";
import MenuList from "@mui/material/MenuList";
import MenuItem from "@mui/material/MenuItem";
import ListItemIcon from "@mui/material/ListItemIcon";
import ClickAwayListener from "@mui/material/ClickAwayListener";

interface MenuItemProps {
  text: string;
  icon?: React.ReactNode;
  endIcon?: React.ReactNode;
  href?: string;
  onClick?: React.MouseEventHandler;
}

interface Props {
  buttonText: string;
  children?: React.ReactNode;
  items?: MenuItemProps[];
}

export default function ButtonDropdown({ buttonText, children, items }: Props) {
  const [isMenuOpen, setIsMenuOpen] = useState(false);

  return (
    <Tooltip
      title={
        children || (
          <div>
            <ClickAwayListener
              onClickAway={() => {
                setIsMenuOpen(false);
              }}
            >
              <MenuList
                sx={{ minWidth: 250, maxWidth: "calc(100vw - 32px)" }}
              >
                {items?.map(item => {
                  return (
                    <MenuItem
                      key={item.href || item.text}
                      sx={{
                        mb: 1,
                        ":last-child": {
                          mb: 0,
                        },
                      }}
                    >
                      <Link
                        href={item.href || "#"}
                        onClick={item.onClick}
                        sx={{
                          display: "flex",
                          alignItems: "center",
                          width: "100%",
                        }}
                      >
                        {item.icon && <ListItemIcon>{item.icon}</ListItemIcon>}
                        <Box component="span" sx={{ flexGrow: 1 }}>
                          {item.text}
                        </Box>
                        {item.endIcon}
                      </Link>
                    </MenuItem>
                  );
                })}
              </MenuList>
            </ClickAwayListener>
          </div>
        )
      }
      placement="bottom-end"
      open={isMenuOpen}
      arrow
    >
      <Button
        variant="contained"
        color="secondary"
        onClick={() => {
          setIsMenuOpen(!isMenuOpen);
        }}
        sx={{ pr: 0.5 }}
      >
        {buttonText}{" "}
        {isMenuOpen ? (
          <ArrowDropUp color="info" sx={{ ml: 0.5 }} />
        ) : (
          <ArrowDropDown color="info" sx={{ ml: 0.5 }} />
        )}
      </Button>
    </Tooltip>
  );
}
