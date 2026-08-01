import { useContext, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import CircularProgress from "@mui/material/CircularProgress";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Table from "@mui/material/Table";
import Head from "@mui/material/TableHead";
import Body from "@mui/material/TableBody";
import Tr from "@mui/material/TableRow";
import Td from "@mui/material/TableCell";
import Chip from "@mui/material/Chip";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import FilteredSearch, {
  formatDateTime,
  toUnixSeconds,
  type FilterDef,
  type FilterValues,
} from "~/components/FilteredSearch";
import { AuthContext } from "~/pages/auth/Auth.context";
import { useSelectedTeam } from "~/layouts/TopMenu/Teams/actions";
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

const METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"];

const STATUS_CODES = [
  "200",
  "204",
  "301",
  "302",
  "304",
  "400",
  "401",
  "403",
  "404",
  "429",
  "500",
  "502",
  "503",
  "504",
];

const FILTER_KEYS = [
  "appId",
  "envId",
  "domainId",
  "hostName",
  "clientIp",
  "path",
  "method",
  "status",
  "isBot",
  "minDurationMs",
  "from",
  "to",
];

interface UseFetchAccessLogsProps {
  query: string;
  cursor?: string;
  refreshToken: number;
}

const useFetchAccessLogs = ({
  query,
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

    const params = new URLSearchParams(query);

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
  }, [query, cursor, refreshToken]);

  return { logs, error, loading, hasNextPage, nextCursor };
};

const formatTs = (ts: string) =>
  ts ? new Date(Number(ts) * 1000).toLocaleString() : "";

export default function AccessLogs() {
  const { teams } = useContext(AuthContext);
  const selectedTeam = useSelectedTeam({ teams });
  const teamId = selectedTeam?.id;
  const [searchParams, setSearchParams] = useSearchParams();
  const [refreshToken, setRefreshToken] = useState(0);

  // The cursor is scoped to the query it was issued for, so changing filters
  // (including via the back button) drops it without an extra render.
  const [cursorState, setCursorState] = useState<{
    query: string;
    value?: string;
  }>();

  const defs = useMemo<FilterDef[]>(
    () => [
      {
        key: "appId",
        label: "App ID",
        kind: "text",
        searchHint: teamId
          ? "Apps in your team — any app ID can still be typed"
          : undefined,
        search: teamId
          ? term =>
              api
                .fetch<{ apps: App[] }>(
                  "/v1/apps?" +
                    new URLSearchParams({ teamId, filter: term }).toString(),
                )
                .then(res =>
                  res.apps.map(app => ({
                    value: app.id,
                    text: `${app.displayName} (${app.id})`,
                  })),
                )
          : undefined,
      },
      { key: "envId", label: "Environment ID", kind: "text" },
      { key: "domainId", label: "Domain ID", kind: "text" },
      { key: "hostName", label: "Host", kind: "text" },
      { key: "clientIp", label: "Client IP", kind: "text" },
      { key: "path", label: "Path", kind: "text" },
      {
        key: "method",
        label: "Method",
        kind: "enum",
        options: METHODS.map(m => ({ value: m, text: m })),
        normalize: v => v.toUpperCase(),
      },
      {
        key: "status",
        label: "Status",
        kind: "enum",
        options: STATUS_CODES.map(s => ({ value: s, text: s })),
      },
      {
        key: "isBot",
        label: "Bots",
        kind: "enum",
        options: [
          { value: "false", text: "Exclude bots" },
          { value: "true", text: "Only bots" },
        ],
        format: v => (v === "true" ? "only" : "excluded"),
      },
      { key: "minDurationMs", label: "Min duration (ms)", kind: "number" },
      { key: "from", label: "From", kind: "datetime", format: formatDateTime },
      { key: "to", label: "To", kind: "datetime", format: formatDateTime },
    ],
    [teamId],
  );

  const values = useMemo<FilterValues>(() => {
    const next: FilterValues = {};

    FILTER_KEYS.forEach(key => {
      const value = searchParams.get(key);

      if (value) {
        next[key] = value;
      }
    });

    return next;
  }, [searchParams]);

  // `from`/`to` are kept in the URL as `datetime-local` strings so they render
  // back into the picker, but the API takes unix seconds.
  const query = useMemo(() => {
    const params = new URLSearchParams();

    Object.entries(values).forEach(([key, value]) => {
      const encoded =
        key === "from" || key === "to" ? toUnixSeconds(value) : value;

      if (encoded) {
        params.set(key, encoded);
      }
    });

    return params.toString();
  }, [values]);

  const cursor = cursorState?.query === query ? cursorState.value : undefined;

  const { logs, error, loading, hasNextPage, nextCursor } = useFetchAccessLogs({
    query,
    cursor,
    refreshToken,
  });

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
      <Box sx={{ px: 4, mb: 4, display: "flex", gap: 2, alignItems: "center" }}>
        <FilteredSearch
          defs={defs}
          values={values}
          placeholder="Filter by app, host, path, status…"
          onChange={next => setSearchParams(new URLSearchParams(next))}
        />
        <Button
          variant="contained"
          disabled={loading}
          sx={{ flexShrink: 0 }}
          onClick={() => setRefreshToken(t => t + 1)}
        >
          {loading ? <CircularProgress size={16} /> : "Refresh"}
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
              onClick={() => setCursorState({ query, value: nextCursor })}
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
