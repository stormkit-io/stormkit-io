import { useEffect, useState } from "react";
import CircularProgress from "@mui/material/CircularProgress";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import TextField from "@mui/material/TextField";
import MenuItem from "@mui/material/MenuItem";
import Table from "@mui/material/Table";
import Head from "@mui/material/TableHead";
import Body from "@mui/material/TableBody";
import Tr from "@mui/material/TableRow";
import Td from "@mui/material/TableCell";
import Chip from "@mui/material/Chip";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import api from "~/utils/api/Api";

interface AccessLogEntry {
  id: string;
  appId: string;
  domainId: string;
  hostName: string;
  requestTimestamp: string;
  method: string;
  path: string;
  statusCode: number;
  clientIp: string;
  userAgent: string;
  referrer: string;
  isBot: boolean;
  bytesSent: number;
  durationMs: number | null;
}

interface Filters {
  appId?: string;
  hostName?: string;
  clientIp?: string;
  path?: string;
  method?: string;
  status?: string;
  isBot?: string;
  minDurationMs?: string;
  from?: string;
  to?: string;
}

interface UseFetchAccessLogsProps {
  filters: Filters;
  cursor?: string;
  refreshToken: number;
}

// The API takes both time bounds as unix seconds, while the inputs produce
// local `datetime-local` strings. Anything unparseable is dropped rather than
// sent as NaN, which the API would read as "no bound".
const toUnixSeconds = (value: string): string => {
  const ms = new Date(value).getTime();

  return isNaN(ms) ? "" : String(Math.floor(ms / 1000));
};

const useFetchAccessLogs = ({
  filters,
  cursor,
  refreshToken,
}: UseFetchAccessLogsProps) => {
  const [logs, setLogs] = useState<AccessLogEntry[]>([]);
  const [nextCursor, setNextCursor] = useState<string>();
  const [hasNextPage, setHasNextPage] = useState(false);
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let unmounted = false;

    setLoading(true);
    setError(undefined);

    const params = new URLSearchParams();

    Object.entries(filters).forEach(([key, value]) => {
      if (!value) {
        return;
      }

      const encoded =
        key === "from" || key === "to" ? toUnixSeconds(value) : value;

      if (encoded) {
        params.set(key, encoded);
      }
    });

    if (cursor) {
      params.set("cursor", cursor);
    }

    api
      .fetch<{
        accessLogs: AccessLogEntry[];
        pagination: { hasNextPage: boolean; cursor?: string };
      }>("/admin/access-logs?" + params.toString())
      .then(res => {
        if (unmounted) {
          return;
        }

        setHasNextPage(res.pagination.hasNextPage);
        setNextCursor(res.pagination.cursor);
        setLogs(cursor ? prev => [...prev, ...res.accessLogs] : res.accessLogs);
      })
      .catch(() => {
        if (!unmounted) {
          setError("Something went wrong while fetching access logs.");
        }
      })
      .finally(() => {
        if (!unmounted) {
          setLoading(false);
        }
      });

    return () => {
      unmounted = true;
    };
  }, [refreshToken, cursor]);

  return { logs, error, loading, hasNextPage, nextCursor };
};

const formatTs = (ts: string) =>
  ts ? new Date(Number(ts) * 1000).toLocaleString() : "";

export default function AccessLogs() {
  const [draft, setDraft] = useState<Filters>({});
  const [filters, setFilters] = useState<Filters>({});
  const [refreshToken, setRefreshToken] = useState(0);
  const [cursor, setCursor] = useState<string>();
  const { logs, error, loading, hasNextPage, nextCursor } = useFetchAccessLogs({
    filters,
    cursor,
    refreshToken,
  });

  const search = () => {
    setCursor(undefined);
    setFilters(draft);
    setRefreshToken(t => t + 1);
  };

  const set = (key: keyof Filters) => (value: string) =>
    setDraft(prev => ({ ...prev, [key]: value }));

  return (
    <Card
      error={error}
      sx={{ backgroundColor: "container.transparent" }}
      info={
        !loading && logs.length === 0
          ? "No access logs match your filters."
          : undefined
      }
      contentPadding={false}
    >
      <CardHeader
        title="Access Logs"
        subtitle="Search raw request logs across all hosted applications. Defaults to the last 24 hours when no time range is set."
      />
      <Box
        sx={{ px: 4, mb: 4, display: "flex", gap: 2, flexWrap: "wrap" }}
        component="form"
        onSubmit={e => {
          e.preventDefault();
          search();
        }}
      >
        <TextField
          variant="filled"
          label="App ID"
          size="small"
          helperText="Recommended — narrows the scan"
          onChange={e => set("appId")(e.target.value)}
        />
        <TextField
          variant="filled"
          label="Host"
          size="small"
          onChange={e => set("hostName")(e.target.value)}
        />
        <TextField
          variant="filled"
          label="Client IP"
          size="small"
          onChange={e => set("clientIp")(e.target.value)}
        />
        <TextField
          variant="filled"
          label="Path"
          size="small"
          onChange={e => set("path")(e.target.value)}
        />
        <TextField
          variant="filled"
          label="Method"
          size="small"
          onChange={e => set("method")(e.target.value.toUpperCase())}
        />
        <TextField
          variant="filled"
          label="Status"
          size="small"
          onChange={e => set("status")(e.target.value)}
        />
        <TextField
          variant="filled"
          label="Bots"
          size="small"
          select
          defaultValue=""
          sx={{ minWidth: 120 }}
          onChange={e => set("isBot")(e.target.value)}
        >
          <MenuItem value="">All</MenuItem>
          <MenuItem value="false">Exclude bots</MenuItem>
          <MenuItem value="true">Only bots</MenuItem>
        </TextField>
        <TextField
          variant="filled"
          label="Min duration (ms)"
          size="small"
          type="number"
          helperText="e.g. 500 for the latency tail"
          onChange={e => set("minDurationMs")(e.target.value)}
        />
        <TextField
          variant="filled"
          label="From"
          size="small"
          type="datetime-local"
          InputLabelProps={{ shrink: true }}
          onChange={e => set("from")(e.target.value)}
        />
        <TextField
          variant="filled"
          label="To"
          size="small"
          type="datetime-local"
          InputLabelProps={{ shrink: true }}
          onChange={e => set("to")(e.target.value)}
        />
        <Button type="submit" variant="contained" disabled={loading}>
          {loading ? <CircularProgress size={16} /> : "Search"}
        </Button>
      </Box>
      <Box sx={{ px: 4 }}>
        <Table size="small">
          <Head>
            <Tr>
              <Td>Time</Td>
              <Td>Method</Td>
              <Td>Status</Td>
              <Td>Duration</Td>
              <Td>Host</Td>
              <Td>Path</Td>
              <Td>Client IP</Td>
              <Td>Bot</Td>
            </Tr>
          </Head>
          <Body>
            {logs.map(log => (
              <Tr key={log.id}>
                <Td>{formatTs(log.requestTimestamp)}</Td>
                <Td>{log.method}</Td>
                <Td>{log.statusCode}</Td>
                <Td sx={{ whiteSpace: "nowrap" }}>
                  {log.durationMs === null ? "—" : `${log.durationMs} ms`}
                </Td>
                <Td>{log.hostName}</Td>
                <Td sx={{ maxWidth: 280, wordBreak: "break-all" }}>
                  {log.path}
                </Td>
                <Td>{log.clientIp}</Td>
                <Td>
                  {log.isBot && (
                    <Chip label="bot" size="small" color="warning" />
                  )}
                </Td>
              </Tr>
            ))}
          </Body>
        </Table>
        {hasNextPage && (
          <Box sx={{ display: "flex", justifyContent: "center", my: 2 }}>
            <Button
              variant="text"
              disabled={loading}
              onClick={() => setCursor(nextCursor)}
            >
              Load more
            </Button>
          </Box>
        )}
      </Box>
      <CardFooter />
    </Card>
  );
}
