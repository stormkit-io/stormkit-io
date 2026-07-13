import { useState } from "react";
import Box from "@mui/material/Box";
import IconButton from "@mui/material/IconButton";
import ArrowRightIcon from "@mui/icons-material/ArrowForwardIos";
import CircularProgress from "@mui/material/CircularProgress";
import LensIcon from "@mui/icons-material/Lens";
import Typography from "@mui/material/Typography";

interface Props {
  logs: Log[];
  isRunning: boolean;
}

const splitLines = (message: string): string[] => {
  // Remove first and last empty lines
  const lines = message
    .replace(/^[\n\s]+|[\n\s]+$/g, "")
    .replace(/\n\n+/g, "\n\n")
    .split("\n");

  return lines;
};

const shouldShowDuration = (
  logs: Log[],
  isRunning: boolean,
  i: number
): boolean => {
  if (isRunning) {
    return (logs.length || 1) - 1 !== i;
  }

  return true;
};

const iconProps = {
  fontSize: 12,
};

export default function DeploymentLogs({ logs, isRunning }: Props) {
  const [isOpen, setIsOpen] = useState<Record<string, boolean>>({});
  const isLastStepAndRunning = (i: number) =>
    isRunning && logs.length - 1 === i;

  return logs?.map(({ title, status, duration, message = "" }, i) => {
    const hasContent = message.trim() !== "";

    return (
      <Box
        key={title}
        data-testid={`deployment-step-${i}`}
        sx={{ px: 3, mb: 2 }}
      >
        <Box
          sx={{
            display: "flex",
            alignItems: "center",
            borderBottom: `1px solid`,
            borderColor: "container.border",
            bgcolor: "container.transparent",
            borderRadius: 2,
            p: 1,
            cursor: hasContent ? "pointer" : "default",
          }}
          onClick={() => {
            hasContent ? setIsOpen({ ...isOpen, [i]: !isOpen[i] }) : undefined;
          }}
        >
          {isLastStepAndRunning(i) ? (
            <CircularProgress
              size={12}
              color="error"
              variant="indeterminate"
              sx={iconProps}
            />
          ) : (
            <LensIcon color={status ? "success" : "error"} sx={iconProps} />
          )}
          <Box
            fontSize="small"
            sx={{
              flex: 1,
              fontFamily: "monospace",
              ml: 2,
              display: "inline-block",
            }}
          >
            {title}
            <IconButton
              sx={{ ml: 0.5, visibility: hasContent ? "visible" : "hidden" }}
              onClick={() => {
                setIsOpen({ ...isOpen, [i]: !isOpen[i] });
              }}
            >
              <ArrowRightIcon
                sx={{
                  fontSize: 12,
                  transform: isOpen[i] ? "rotate(90deg)" : "",
                }}
              />
            </IconButton>
          </Box>
          {shouldShowDuration(logs, isRunning, i) ? (
            <Typography
              component="span"
              sx={{ color: "text.secondary", fontSize: 11 }}
            >
              {duration}s
            </Typography>
          ) : (
            ""
          )}
        </Box>
        {(isOpen[i] || isLastStepAndRunning(i)) && (
          <Box
            component="code"
            bgcolor="transparent"
            sx={{
              fontFamily: "monospace",
              fontSize: 12,
              py: 2,
              lineHeight: 1.5,
              overflow: "auto",
              display: "block",
              whiteSpace: "pre",
            }}
            style={{ maxHeight: "400px" }}
          >
            {splitLines(message).map((line, i) => (
              <Box
                key={i}
                sx={{
                  display: "flex",
                  color: "text.secondary",
                  "&:hover": { bgcolor: "rgba(0,0,0,0.1)" },
                }}
              >
                <Box
                  component="span"
                  sx={{
                    display: "inline-block",
                    minWidth: "50px",
                    maxWidth: "50px",
                    width: "100%",
                    textAlign: "right",
                    mr: 1,
                  }}
                >
                  {i + 1}.
                </Box>{" "}
                {line}
              </Box>
            ))}
          </Box>
        )}
      </Box>
    );
  });
}
