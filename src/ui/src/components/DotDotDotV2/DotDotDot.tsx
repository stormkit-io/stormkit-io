import React, { useState } from "react";
import cn from "classnames";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Tooltip from "@mui/material/Tooltip";
import IconButton from "@mui/material/IconButton";
import MoreHoriz from "@mui/icons-material/MoreHoriz";
import ClickAwayListener from "@mui/material/ClickAwayListener";

export interface Item {
  text: React.ReactNode;
  href?: string;
  icon?: string | React.ReactNode;
  className?: string;
  disabled?: boolean;
  hidden?: boolean;
  onClick?: (close?: () => void) => boolean | void;
}

interface Props {
  items: Item[];
  label?: string;
}

export default function DotDotDot({ label, items }: Props) {
  const [isOpen, toggleIsOpen] = useState(false);

  return (
    <Tooltip
      placement="bottom-end"
      open={isOpen}
      arrow
      title={
        <ClickAwayListener onClickAway={() => toggleIsOpen(false)}>
          <Box
            sx={{
              display: "flex",
              flexDirection: "column",
              alignItems: "flex-start",
              minWidth: "185px",
              maxWidth: "calc(100vw - 32px)",
            }}
          >
            {items.map((item, index) => (
              <Box
                key={index}
                sx={{
                  display: item.hidden ? "none" : undefined,
                  width: "100%",
                  borderBottom: `1px solid rgba(255,255,255,0.1)`,
                  cursor: item.disabled ? "not-allowed" : undefined,
                  ":last-of-type": {
                    borderBottom: "none",
                  },
                }}
              >
                <Button
                  variant="text"
                  color="info"
                  href={item.href}
                  disabled={item.disabled}
                  fullWidth
                  size="large"
                  sx={{
                    justifyContent: "flex-start",
                    opacity: item.disabled ? 0.25 : 1,
                    backgroundColor: "transparent !important",
                  }}
                  startIcon={
                    typeof item.icon === "string" ? (
                      <Box
                        component="span"
                        sx={{ fontSize: "15px !important" }}
                        className={cn(item.icon, "fa-fw")}
                      />
                    ) : typeof item.icon === "object" ? (
                      item.icon
                    ) : undefined
                  }
                  onClick={() => {
                    if (item.onClick?.(() => toggleIsOpen(false)) !== false) {
                      toggleIsOpen(false);
                    }
                  }}
                >
                  {item.text}
                </Button>
              </Box>
            ))}
          </Box>
        </ClickAwayListener>
      }
    >
      <IconButton
        onClick={() => toggleIsOpen(!isOpen)}
        sx={{ position: "relative", right: -10 }}
        aria-label={label}
      >
        <MoreHoriz sx={{ fontSize: 16 }} aria-label="expand" />
      </IconButton>
    </Tooltip>
  );
}
