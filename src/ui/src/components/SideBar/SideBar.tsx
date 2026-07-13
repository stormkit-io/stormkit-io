import React, { useEffect, useState } from "react";
import Box from "@mui/material/Box";
import { createPortal } from "react-dom";
import "./SideBar.css";

interface Props {
  isOpen: boolean;
  maxWidth?: "max-w-128" | "max-w-72";
  children: React.ReactNode;
}

const SideBar: React.FC<Props> = ({ isOpen, children }) => {
  const [marginTop, setMarginTop] = useState<number>(60);

  useEffect(() => {
    if (isOpen) {
      setMarginTop(
        document.querySelector("#top-bar")?.getBoundingClientRect().height || 60
      );
    }
  }, [isOpen]);

  return createPortal(
    <Box
      boxShadow={6}
      bgcolor="background.paper"
      sx={{
        overflow: "auto",
        display: isOpen ? "block" : "none",
        position: "fixed",
        bottom: 0,
        right: 0,
        top: `${marginTop}px`,
        width: "100%",
        pl: { md: 0.5 },
        maxWidth: { lg: 500 },
        animation: isOpen ? "slideFromRight 200ms ease-in-out forwards" : "",
      }}
    >
      {children}
    </Box>,
    document.querySelector("#side-bar-root")!
  );
};

export default SideBar;
