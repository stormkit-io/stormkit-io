import { useContext, useState } from "react";
import { useParams } from "react-router";
import Typography from "@mui/material/Typography";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Drawer from "@mui/material/Drawer";
import RefreshOutlined from "@mui/icons-material/RefreshOutlined";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import { AppContext } from "~/pages/apps/[id]/App.context";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import Span from "~/components/Span";
import CardRow from "~/components/CardRow";
import EmptyList from "~/components/EmptyPage";
import { formatDate } from "~/utils/helpers/date";
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
            setDrawerContent(log);
          }}
        >
          <Typography sx={{ display: "flex", alignItems: "center" }}>
            <Span
              color={
                log.response?.code?.toString()?.[0] === "2"
                  ? "success"
                  : log.response?.code
                    ? "default"
                    : "failure"
              }
            >
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
      <Drawer
        anchor="right"
        open={Boolean(drawerContent)}
        onClose={() => setDrawerContent(undefined)}
      >
        {drawerContent && (
          <Card sx={{ minWidth: "40vw", maxWidth: "600px", margin: "0" }}>
            <CardHeader
              title="Log details"
              subtitle={formatDate(drawerContent!.createdAt * 1000)}
            />
            <Box sx={{ fontSize: 12 }}>
              <Box sx={{ mb: 4 }}>
                <Typography variant="h3" sx={{ mb: 2 }}>
                  Request payload
                </Typography>
                <Box
                  component="pre"
                  sx={{
                    bgcolor: "container.transparent",
                    p: 2,
                    maxWidth: "100%",
                    overflow: "auto",
                  }}
                >
                  {drawerContent?.request?.payload || "No payload"}
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
                      drawerContent.response?.code?.toString()?.[0] === "2"
                        ? "success"
                        : undefined
                    }
                  >
                    {drawerContent.response?.code || "ERR"}
                  </Span>
                </Typography>
                <Box
                  component="pre"
                  sx={{
                    bgcolor: "container.transparent",
                    p: 2,
                    maxWidth: "100%",
                    overflow: "auto",
                  }}
                >
                  {drawerContent?.response?.body ||
                  drawerContent?.response?.error ||
                  "No response body"}
                </Box>
              </Box>
            </Box>
          </Card>
        )}
      </Drawer>
    </Card>
  );
}
