import Typography from "@mui/material/Typography";
import Span from "~/components/Span";
import CardRow from "~/components/CardRow";
import EmptyList from "~/components/EmptyPage";
import { formatDate } from "~/utils/helpers/date";
import { statusColor } from "./statusColor";

interface Props {
  logs?: TriggerLog[];
  onSelect: (log: TriggerLog) => void;
}

export default function TriggerLogList({ logs, onSelect }: Props) {
  return (
    <>
      {logs?.map(log => (
        <CardRow
          key={log.createdAt}
          sx={{
            cursor: "pointer",
            "&:hover": {
              backgroundColor: "container.transparent",
            },
          }}
          data-testid="trigger-log"
          onClick={() => {
            onSelect(log);
          }}
        >
          <Typography sx={{ display: "flex", alignItems: "center" }}>
            <Span color={statusColor(log.response?.code)}>
              {log.response?.code || "ERR"}
            </Span>
            <Span>{log.request?.url}</Span>
            <Typography
              component="span"
              color="text.secondary"
              sx={{ flex: 1, textAlign: "right" }}
            >
              {formatDate(log.createdAt * 1000)}
            </Typography>
          </Typography>
        </CardRow>
      ))}
      {logs?.length === 0 && <EmptyList />}
    </>
  );
}
