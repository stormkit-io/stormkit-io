import { useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Drawer from "@mui/material/Drawer";
import ArrowBackIcon from "@mui/icons-material/ArrowBack";
import RefreshOutlined from "@mui/icons-material/RefreshOutlined";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import { formatDate } from "~/utils/helpers/date";
import { useFetchTriggerLogs } from "../actions";
import TriggerLogDetails from "./TriggerLogDetails";
import TriggerLogList from "./TriggerLogList";

interface Props {
  trigger: FunctionTrigger;
  appId: string;
  envId: string;
  initialLog?: TriggerLog;
  open: boolean;
  onClose: () => void;
  /** Fires once the slide-out transition has finished. */
  onExited: () => void;
}

const cardStyles = {
  minWidth: "40vw",
  maxWidth: "600px",
  margin: "0",
  border: "none",
  borderRadius: 0,
};

export default function TriggerLogsDrawer({
  trigger,
  appId,
  envId,
  initialLog,
  open,
  onClose,
  onExited,
}: Props) {
  const [selected, setSelected] = useState<TriggerLog | undefined>(initialLog);
  const [refreshToken, setRefreshToken] = useState(0);
  // Opening the drawer on a fresh invoke shows the details straight away, so
  // the list is only fetched once the user actually navigates back to it.
  const [listRequested, setListRequested] = useState(!initialLog);
  const { logs, error, loading } = useFetchTriggerLogs({
    triggerId: trigger.id!,
    appId,
    envId,
    refreshToken,
    enabled: listRequested,
  });

  return (
    <Drawer
      anchor="right"
      open={open}
      onClose={onClose}
      onTransitionExited={onExited}
    >
      {selected ? (
        <Card sx={cardStyles}>
          <CardHeader
            title="Log details"
            subtitle={formatDate(selected.createdAt * 1000)}
            actions={
              <Button
                type="button"
                sx={{ mr: 0 }}
                variant="text"
                color="info"
                data-testid="back-to-logs"
                onClick={() => {
                  setListRequested(true);
                  setSelected(undefined);
                }}
              >
                <ArrowBackIcon fontSize="small" />
              </Button>
            }
          />
          <TriggerLogDetails log={selected} />
        </Card>
      ) : (
        <Card
          sx={cardStyles}
          loading={loading}
          error={error}
          contentPadding={false}
        >
          <CardHeader
            title="Past triggers"
            subtitle={trigger.options.url}
            actions={
              <Button
                type="button"
                sx={{ mr: 0 }}
                variant="text"
                color="info"
                data-testid="refresh-logs"
                onClick={() => setRefreshToken(Date.now())}
              >
                <RefreshOutlined fontSize="small" />
              </Button>
            }
          />
          <Box>
            <TriggerLogList logs={logs} onSelect={setSelected} />
          </Box>
        </Card>
      )}
    </Drawer>
  );
}
