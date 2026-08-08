import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Span from "~/components/Span";
import Markdown from "~/components/Markdown";
import CodeBlock from "~/components/CodeBlock";
import { formatBody } from "./prettyPrint";
import { statusColor } from "./statusColor";

interface Props {
  log: TriggerLog;
  documentation?: string;
}

const sectionStyles = {
  bgcolor: "container.transparent",
  p: 2,
  maxWidth: "100%",
  overflow: "auto",
  wordBreak: "break-word",
};

export default function TriggerLogDetails({ log, documentation }: Props) {
  return (
    <Box sx={{ fontSize: 12 }}>
      <Box sx={{ mb: 4 }}>
        <Typography variant="h3" sx={{ mb: 2 }}>
          Request payload
        </Typography>
        <CodeBlock>{formatBody(log.request?.payload, "No payload")}</CodeBlock>
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
            color={statusColor(log.response?.code)}
          >
            {log.response?.code || "ERR"}
          </Span>
        </Typography>
        <CodeBlock>
          {formatBody(
            log.response?.body || log.response?.error,
            "No response body"
          )}
        </CodeBlock>
      </Box>
      {documentation && (
        <Box sx={{ mt: 4 }}>
          <Typography variant="h3" sx={{ mb: 2 }}>
            Documentation
          </Typography>
          <Box sx={sectionStyles}>
            <Markdown sx={{ fontSize: 12 }}>{documentation}</Markdown>
          </Box>
        </Box>
      )}
    </Box>
  );
}
