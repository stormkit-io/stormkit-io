import React from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";

interface Props {
  withLogo?: boolean;
  children?: React.ReactNode;
}

export default function Error404({ children }: Props) {
  return (
    <Box
      sx={{
        display: "flex",
        height: "100%",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        mx: "auto",
        minHeight: "50vh",
      }}
      maxWidth="lg"
    >
      <Box sx={{ textAlign: "center" }}>
        <Typography
          color="secondary"
          sx={{ fontSize: 120, fontWeight: "bold" }}
        >
          4 oh 4
        </Typography>
        <Typography component="div" sx={{ fontSize: 28, lineHeight: 1 }}>
          {children || "There is nothing under this link"}
        </Typography>
      </Box>
    </Box>
  );
}
