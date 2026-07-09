import type { SxProps } from "@mui/material";
import React from "react";
import { grey } from "@mui/material/colors";
import Box from "@mui/material/Box";
import MuiModal, { ModalProps } from "@mui/material/Modal";

interface Props extends ModalProps {
  sx?: SxProps;
  height?: string;
  maxWidth?: "sm" | "md" | "lg" | string;
}

const Modal: React.FC<Props> = ({
  children,
  open,
  maxWidth = "md",
  height,
  ...rest
}) => {
  return (
    <MuiModal
      open={open}
      {...rest}
      sx={{
        display: "flex",
        width: "100vw",
        height: "100vh",
        alignItems: "center",
        justifyContent: "center",
        ...rest.sx,
      }}
    >
      <Box
        maxWidth={maxWidth}
        width="100%"
        bgcolor="background.default"
        sx={{
          borderRadius: 1,
          height,
          maxHeight: "90vh",
          mx: { xs: 2, md: 0 },
          overflow: "auto",
          border: `1px solid ${grey[900]}`,
        }}
      >
        {children}
      </Box>
    </MuiModal>
  );
};

export default Modal;
