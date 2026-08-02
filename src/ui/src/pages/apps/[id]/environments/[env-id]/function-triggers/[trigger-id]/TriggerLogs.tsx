import { useContext, useState } from "react";
import { useParams } from "react-router";
import Button from "@mui/material/Button";
import Drawer from "@mui/material/Drawer";
import RefreshOutlined from "@mui/icons-material/RefreshOutlined";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import { AppContext } from "~/pages/apps/[id]/App.context";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import { formatDate } from "~/utils/helpers/date";
import TriggerLogDetails from "../_components/TriggerLogDetails";
import TriggerLogList from "../_components/TriggerLogList";
import { useFetchTriggerLogs } from "../actions";

export default function TriggerLogs() {
  const [refreshToken, setRefreshToken] = useState(0);
  const [drawerContent, setDrawerContent] = useState<TriggerLog>();
  const { triggerId } = useParams();
  const { app } = useContext(AppContext);
  const { environment } = useContext(EnvironmentContext);
  const { logs, error, loading } = useFetchTriggerLogs({
    triggerId: triggerId!,
    appId: app.id,
    envId: environment.id!,
    refreshToken,
  });

  return (
    <Card
      loading={loading}
      error={error}
      contentPadding={false}
      sx={{
        width: "100%",
      }}
    >
      <CardHeader
        title="Trigger logs"
        subtitle="Last 25 logs for this trigger"
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
      <TriggerLogList logs={logs} onSelect={setDrawerContent} />
      <Drawer
        anchor="right"
        open={Boolean(drawerContent)}
        onClose={() => setDrawerContent(undefined)}
      >
        {drawerContent && (
          <Card sx={{ minWidth: "40vw", maxWidth: "600px", margin: "0" }}>
            <CardHeader
              title="Log details"
              subtitle={formatDate(drawerContent.createdAt * 1000)}
            />
            <TriggerLogDetails log={drawerContent} />
          </Card>
        )}
      </Drawer>
    </Card>
  );
}
