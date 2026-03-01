import type { BoxProps } from "@mui/material";
import type { Item } from "~/components/DotDotDotV2/DotDotDot";
import React, { useContext } from "react";
import Box from "@mui/material/Box";
import Chip from "@mui/material/Chip";
import DotDotDot from "~/components/DotDotDotV2";
import CardContext from "../Card/Card.context";

interface Props extends BoxProps {
  menuItems?: Item[];
  menuLabel?: string;
  icon?: React.ReactNode;
  chipLabel?: React.ReactNode;
  chipColor?: "success" | "info" | "warning";
  chipSx?: BoxProps["sx"];
  actions?: React.ReactNode;
}

export default function CardRow({
  sx,
  children,
  menuItems,
  menuLabel = "expand",
  chipLabel,
  chipColor,
  actions,
  icon,
  chipSx,
  ...rest
}: Props) {
  const { size } = useContext(CardContext);
  const space = size === "medium" ? 4 : 2;

  return (
    <Box
      sx={{
        borderBottom: "1px solid",
        borderColor: "container.border",
        display: "flex",
        alignItems: "center",
        px: space,
        py: 2,
        ":first-of-type": {
          borderTop: `1px solid`,
          borderColor: "container.border",
        },
        ":last-child": {
          borderBottom: "none",
        },
        ...sx,
      }}
      {...rest}
    >
      {icon && <Box sx={{ mr: 2 }}>{icon}</Box>}
      <Box sx={{ flex: 1 }}>{children}</Box>
      {chipLabel && (
        <Chip
          color={chipColor}
          label={chipLabel}
          size="small"
          sx={{ ml: 1, mr: menuItems ? 2 : 0, ...chipSx }}
        />
      )}
      {(menuItems || actions) && (
        <Box
          sx={{
            display: "flex",
            alignItems: "flex-end",
            justifyContent: "space-between",
            flexDirection: "column",
          }}
        >
          {menuItems && (
            <Box
              sx={{ visibility: menuItems.length === 0 ? "hidden" : "visible" }}
            >
              <DotDotDot label={menuLabel} items={menuItems} />
            </Box>
          )}
          {actions}
        </Box>
      )}
    </Box>
  );
}
