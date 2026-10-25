import type { AnalyticsRow, TimeSpan } from "./index.d";
import { useState } from "react";
import Box from "@mui/material/Box";
import Alert from "@mui/material/Alert";
import Typography from "@mui/material/Typography";
import IconButton from "@mui/material/IconButton";
import TextField from "@mui/material/TextField";
import LinearProgress from "@mui/material/LinearProgress";
import ArrowForward from "@mui/icons-material/ArrowForwardIos";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardRow from "~/components/CardRow";
import CardFooter from "~/components/CardFooter";
import { useFetchAnalyticsReport } from "./actions";
import { channelLabels, spanText, truncate } from "./helpers";
import { formatDuration } from "./Visitors";

const formatter = new Intl.NumberFormat();

// otherPath is the row that single-view pages are folded into on busy days.
const otherPath = "(other)";

// PathStats renders a path's page views, visitors and time on page.
function PathStats({ row }: { row: AnalyticsRow }) {
  return (
    <Typography component="span" sx={{ fontSize: 13 }}>
      {formatter.format(row.pageViews || 0)} views
      {row.visitors !== undefined &&
        ` · ${formatter.format(row.visitors)} visitors`}
      {row.avgTimeOnPage !== undefined &&
        ` · ${formatDuration(row.avgTimeOnPage)}`}
    </Typography>
  );
}

interface PathChannelsProps {
  envId: string;
  domainId?: string;
  path: string;
  ts: TimeSpan;
}

// PathChannels answers who read a page and where they came from.
function PathChannels({ envId, domainId, path, ts }: PathChannelsProps) {
  const { report, loading, error } = useFetchAnalyticsReport({
    envId,
    domainId,
    query: { metric: "channels", ts, path },
  });

  if (loading) {
    return <LinearProgress color="secondary" />;
  }

  if (error) {
    return (
      <Alert color="error" sx={{ m: 2 }}>
        {error}
      </Alert>
    );
  }

  const rows = report?.rows || [];

  return (
    <>
      <CardRow>
        <PathStats row={report?.totals || {}} />
      </CardRow>
      {rows.length ? (
        rows.map(row => (
          <CardRow
            key={row.channel}
            chipLabel={
              <Typography component="span">
                {formatter.format(row.visits || 0)}
              </Typography>
            }
          >
            <Typography component="span">
              {channelLabels[row.channel!] || row.channel}
            </Typography>
          </CardRow>
        ))
      ) : (
        <Alert
          color="info"
          sx={{
            mb: 4,
            px: 4,
            bgcolor: "rgba(255,255,255,0.025)",
            borderRadius: 0,
          }}
        >
          <Box>No visits found for this path.</Box>
        </Alert>
      )}
    </>
  );
}

interface Props {
  environment: Environment;
  domain: Domain;
  ts: TimeSpan;
}

// normalizePrefix makes a typed prefix a path, as the API requires.
const normalizePrefix = (value: string) => {
  const prefix = value.trim();

  return prefix && !prefix.startsWith("/") ? `/${prefix}` : prefix;
};

export default function TopPaths({ environment, domain, ts }: Props) {
  const envId = environment.id!;
  const [prefixInput, setPrefixInput] = useState("");
  const [pathPrefix, setPathPrefix] = useState("");

  // A drill-down belongs to the domain it was opened on, so picking another
  // domain returns to the list.
  const [drillDown, setDrillDown] = useState({
    domainId: domain?.id,
    path: "",
  });

  // Reset during render rather than in an effect, so the old drill-down never
  // renders (and requests) for the new domain, nor returns with the old one.
  if (drillDown.domainId !== domain?.id) {
    setDrillDown({ domainId: domain?.id, path: "" });
  }

  const requestPath = drillDown.domainId === domain?.id ? drillDown.path : "";
  const setRequestPath = (path: string) =>
    setDrillDown({ domainId: domain?.id, path });

  const applyPrefix = () => {
    const prefix = normalizePrefix(prefixInput);

    if (prefix !== pathPrefix) {
      setPathPrefix(prefix);
      setRequestPath("");
    }
  };

  const { report, error, loading } = useFetchAnalyticsReport({
    envId,
    domainId: domain?.id,
    query: { metric: "paths", ts, pathPrefix, limit: 50 },
  });

  const paths = report?.rows || [];

  return (
    <Card
      sx={{
        width: { xs: "100%", md: "50%" },
        margin: "",
        display: "flex",
        flexDirection: "column",
      }}
      error={error}
      loading={loading && !report}
    >
      <CardHeader
        title="Paths"
        subtitle={`Top visited paths ${spanText(ts)}.`}
        actions={
          <TextField
            size="small"
            variant="filled"
            label="Path prefix"
            placeholder="/blog/"
            value={prefixInput}
            onChange={e => setPrefixInput(e.target.value)}
            onKeyDown={e => {
              if (e.key === "Enter") {
                applyPrefix();
              }
            }}
            onBlur={applyPrefix}
            sx={{ maxWidth: 160 }}
          />
        }
      />
      {loading && report && <LinearProgress color="secondary" />}
      <Box
        sx={{
          maxHeight: "300px",
          overflow: "auto",
        }}
      >
        {!requestPath ? (
          !report ? null : paths.length ? (
            paths.map(path => (
              <CardRow
                key={path.path}
                chipLabel={<PathStats row={path} />}
                actions={
                  // (other) groups folded pages; it is not a path to look
                  // into.
                  path.path === otherPath ? undefined : (
                    <IconButton
                      sx={{ ml: 2 }}
                      size="small"
                      aria-label={`Show where ${path.path} visitors came from`}
                      onClick={() => {
                        setRequestPath(path.path!);
                      }}
                    >
                      <ArrowForward sx={{ fontSize: 12 }} />
                    </IconButton>
                  )
                }
              >
                <Typography component="span">{truncate(path.path!)}</Typography>
              </CardRow>
            ))
          ) : (
            <Alert
              color="info"
              sx={{ mx: 2, mb: 2, bgcolor: "rgba(255,255,255,0.025)" }}
            >
              No page views {spanText(ts)}
              {pathPrefix ? ` under ${pathPrefix}` : ""}.
            </Alert>
          )
        ) : (
          <Box sx={{ flex: 1 }}>
            <CardRow
              actions={
                <IconButton
                  sx={{ ml: 1, textAlign: "center" }}
                  aria-label="Back to paths"
                  onClick={() => {
                    setRequestPath("");
                  }}
                >
                  <ArrowForward
                    sx={{ fontSize: 16, transform: "rotate(180deg)" }}
                  />
                </IconButton>
              }
            >
              <Typography component="span">{requestPath}</Typography>
            </CardRow>
            <PathChannels
              envId={envId}
              domainId={domain?.id}
              path={requestPath}
              ts={ts}
            />
          </Box>
        )}
      </Box>
      <CardFooter sx={{ textAlign: "left" }}>
        <Typography sx={{ fontSize: 12 }}>
          Time on page needs client-side tracking. (other) groups single-view
          pages on busy days.
        </Typography>
      </CardFooter>
    </Card>
  );
}
