import type { SxProps } from "@mui/material";
import React from "react";
import Typography from "@mui/material/Typography";

type Color = "success" | "warning" | "failure" | "default";

interface Props {
  children: React.ReactNode;
  color?: Color;
  sx?: SxProps;
  size?: "small" | "medium" | "large";
}

const colors: Record<Color, string> = {
  success: "success.main",
  warning: "warning",
  failure: "error",
  default: "container.transparent",
};

export default function InlineBox({
  children,
  color = "default",
  size = "medium",
  sx,
}: Props) {
  return (
    <Typography
      component="span"
      bgcolor={colors[color]}
      sx={{
        py: size === "medium" ? 0.5 : size === "large" ? 1 : 0.375,
        px: size === "medium" ? 1.5 : size === "large" ? 3 : 1,
        mr: size === "medium" ? 1.5 : size === "large" ? 3 : 1,
        borderRadius: 2,
        display: "inline-flex",
        alignItems: "center",
        verticalAlign: "middle",
        fontSize: size === "medium" ? "inherit" : size === "large" ? 16 : 10,
        ...sx,
      }}
    >
      {children}
    </Typography>
  );
}
