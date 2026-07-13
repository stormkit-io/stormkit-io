import type { BoxProps } from "@mui/material";
import React, { useState, useEffect } from "react";
import Alert from "@mui/material/Alert";
import AlertTitle from "@mui/material/AlertTitle";
import Box from "@mui/material/Box";
import ReportIcon from "@mui/icons-material/Report";
import LinearProgress from "@mui/material/LinearProgress";
import CardHeader from "../CardHeader";
import CardFooter from "../CardFooter";
import CardRow from "../CardRow";
import CardContext, { Size } from "./Card.context";

interface Props extends BoxProps {
  successTitle?: string | false;
  errorTitle?: string | false;
  error?: React.ReactNode | string[];
  info?: React.ReactNode;
  success?: React.ReactNode;
  contentPadding?: boolean; // When false, the content will not have padding x.
  loading?: boolean;
  size?: Size;
  onSuccessClose?: () => void;
  onErrorClose?: () => void;
}

function Card({
  sx,
  errorTitle = "Error",
  successTitle = "Success",
  error,
  success,
  info,
  children,
  loading,
  contentPadding = true,
  size = "medium",
  onSuccessClose,
  onErrorClose,
  ...rest
}: Props) {
  let header;
  let footer;
  let content: React.ReactNode[] = [];
  let isCardRow = false;
  const [isLoadingFirstTime, setIsLoadingFirstTime] = useState<boolean>();

  useEffect(() => {
    if (typeof isLoadingFirstTime === "undefined" || isLoadingFirstTime) {
      setIsLoadingFirstTime(loading);
    }
  }, [loading]);

  React.Children.forEach(children, child => {
    if (!React.isValidElement(child)) {
      return;
    }

    if (child.type === CardHeader) {
      header = child;
    } else if (child.type === CardFooter) {
      footer = child;
    } else {
      // @ts-ignore
      const { children } = child.props || {};

      if (Array.isArray(children) && children[0]?.type === CardRow) {
        isCardRow = true;
      }

      content.push(child);
    }
  });

  const p = size === "small" ? 2 : 4;

  return (
    <CardContext.Provider value={{ size }}>
      <Box
        sx={{
          m: "auto",
          bgcolor: "background.paper",
          border: `1px solid`,
          borderColor: "container.border",
          borderRadius: 1,
          position: "relative",
          pt: !header && content.length ? p : 0,
          pb: !footer && content.length ? p : 0,
          ...sx,
        }}
        {...rest}
      >
        {header}
        {content && !isLoadingFirstTime && (
          <Box
            sx={{
              px: isCardRow || !contentPadding ? 0 : p,
              flex: 1,
              width: "100%",
            }}
          >
            {content}
          </Box>
        )}
        {loading && (
          <Box
            data-testid="card-loading"
            sx={{
              position: "absolute",
              left: 0,
              right: 0,
              flex: 1,
            }}
          >
            <LinearProgress color="secondary" />
          </Box>
        )}
        {!loading && error && (
          <Alert
            color="error"
            sx={{ mb: footer ? p : 0, px: p, mx: p }}
            onClose={onErrorClose}
          >
            {errorTitle && <AlertTitle>{errorTitle}</AlertTitle>}
            {Array.isArray(error) ? (
              <Box component="ul" sx={{ m: 0, pl: 2, listStyleType: "disc" }}>
                {error.map((e, i) => (
                  <Box component="li" key={i}>
                    {e}
                  </Box>
                ))}
              </Box>
            ) : (
              <Box>{error}</Box>
            )}
          </Alert>
        )}
        {!loading && success && (
          <Alert
            color="success"
            sx={{ mb: footer ? p : 0, px: p, mx: p }}
            onClose={onSuccessClose}
          >
            {successTitle && <AlertTitle>{successTitle}</AlertTitle>}
            <Box sx={{ flex: 1 }}>{success}</Box>
          </Alert>
        )}
        {!loading && info && (
          <Alert
            color="info"
            icon={<ReportIcon />}
            sx={{
              display: "flex",
              alignItems: "center",
              mb: footer ? p : 0,
              px: p,
              mx: p,
            }}
          >
            {info}
          </Alert>
        )}
        {footer && (
          <Box
            sx={{
              px: p,
              py: 2,
              pt: isLoadingFirstTime ? p : 2,
              borderTop: "1px solid",
              borderColor: "container.border",
            }}
          >
            {footer}
          </Box>
        )}
      </Box>
    </CardContext.Provider>
  );
}

export default Card;
