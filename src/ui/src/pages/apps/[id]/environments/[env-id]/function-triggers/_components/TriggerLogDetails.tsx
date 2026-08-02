import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Span from "~/components/Span";
import { formatBody } from "./prettyPrint";

interface Props {
  log: TriggerLog;
}

const preStyles = {
  bgcolor: "container.transparent",
  p: 2,
  maxWidth: "100%",
  overflow: "auto",
  whiteSpace: "pre-wrap",
  wordBreak: "break-word",
};

export default function TriggerLogDetails({ log }: Props) {
  return (
    <Box sx={{ fontSize: 12 }}>
      <Box sx={{ mb: 4 }}>
        <Typography variant="h3" sx={{ mb: 2 }}>
          Request payload
        </Typography>
        <Box component="pre" sx={preStyles}>
          {formatBody(log.request?.payload, "No payload")}
        </Box>
      </Box>
      <Box>
        <Typography
          variant="h3"
          sx={{
            mb: 2,
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
          }}
        >
          <Box component="span">Response body</Box>
          <Span
            sx={{ mr: 0 }}
            size="small"
            color={
              log.response?.code?.toString()?.[0] === "2"
                ? "success"
                : undefined
            }
          >
            {log.response?.code || "ERR"}
          </Span>
        </Typography>
        <Box component="pre" sx={preStyles}>
          {formatBody(
            log.response?.body || log.response?.error,
            "No response body"
          )}
        </Box>
      </Box>
    </Box>
  );
}
