import type { Log } from "./actions";
import React, { useContext, useState } from "react";
import { useParams, useLocation } from "react-router";
import Button from "@mui/material/Button";
import Link from "@mui/material/Link";
import Box from "@mui/material/Box";
import Switch from "@mui/material/Switch";
import FormControlLabel from "@mui/material/FormControlLabel";
import { grey } from "@mui/material/colors";
import ArrowDownwardIcon from "@mui/icons-material/ArrowDownward";
import SortIcon from "@mui/icons-material/SwapVert";
import EmptyPage from "~/components/EmptyPage";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import { AppContext } from "~/pages/apps/[id]/App.context";
import { useFetchDeploymentRuntimeLogs } from "./actions";

const renderLog = (log: Log, i: number) => {
  let data = log.data.split(/END\sRequestId:/)[0];

  // Regex to remove lines like:
  // 2022-10-20T13:05:30.027Z	undefined	TRACE
  // 2022-10-20T13:05:30.028Z	undefined	ERROR
  data = data.replace(
    /20[0-9]{2}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[[0-9]+Z.*(ERROR|TRACE|INFO|WARN)\t?/g,
    ""
  );

  // Regex to remove lines like:
  // START RequestId: <request-id> Version: 1
  data = data.replace(/START\sRequestId:\s[\sa-z0-9-]+\sVersion:\s[0-9]+/, "");
  data = data.trim();

  if (data.length === 0) {
    return null;
  }

  let date;

  if (log.timestamp?.length === 10) {
    date = new Date(Number(log.timestamp) * 1000);
  } else if (log.timestamp?.length === 13) {
    date = new Date(Number(log.timestamp));
  } else {
    date = new Date();
  }

  const isoDateTime = new Date(
    date.getTime() - date.getTimezoneOffset() * 60000
  ).toISOString();

  return (
    <Box
      key={`${log.id}-${i}`}
      sx={{
        mb: 1,
        px: 2,
        py: 0.5,
        mx: 0,
        ":hover": { backgroundColor: "container.transparent" },
        ":first-of-type": { mt: 1.5 },
        ":last-child": { mb: 1.5 },
        display: "flex",
      }}
    >
      <Box sx={{ color: "text.secondary" }}>{i}.</Box>
      <Box sx={{ flex: 1, px: 2, pr: 8 }}>{data}</Box>
      <Box sx={{ color: "text.secondary" }}>
        {isoDateTime.split(".")[0].replace("T", " ")}
      </Box>
    </Box>
  );
};

const RuntimeLogs: React.FC = () => {
  const location = useLocation();
  const [whitespace, setWhiteSpace] = useState(true);
  const [sort, setSort] = useState<"asc" | "desc">("desc");
  const { deploymentId } = useParams();
  const { app } = useContext(AppContext);
  const [resetLogs, setResetLogs] = useState(true);
  const [keySetId, setKeySeyId] = useState<string>();

  const { logs, error, loading, hasNextPage } = useFetchDeploymentRuntimeLogs({
    appId: app.id,
    deploymentId: deploymentId!,
    keySetId,
    sort,
    reset: resetLogs,
  });

  const isLoadingFirstPage = resetLogs && loading && logs?.length === 0;

  return (
    <Card sx={{ width: "100%" }} loading={isLoadingFirstPage} error={error}>
      <CardHeader
        title="Runtime logs"
        subtitle={
          <>
            Logs produced by server side rendered apps and API functions will be
            displayed here. <br /> These logs belong to deployment{" "}
            <Link href={location.pathname.replace("/runtime-logs", "")}>
              #{deploymentId}
            </Link>
            .
          </>
        }
      />
      {logs?.length > 0 ? (
        <>
          <Box sx={{ display: "flex", justifyContent: "space-between" }}>
            <FormControlLabel
              sx={{ pl: 0, ml: 0 }}
              label="Preserve whitespace"
              control={
                <Switch
                  color="secondary"
                  checked={whitespace}
                  onChange={e => {
                    setWhiteSpace(e.target.checked);
                  }}
                />
              }
              labelPlacement="start"
            />
            <Box>
              <Button
                variant="outlined"
                color="primary"
                startIcon={<SortIcon />}
                sx={{ mr: 2 }}
                onClick={() => {
                  setSort(sort === "asc" ? "desc" : "asc");
                  setResetLogs(true);
                  setKeySeyId(undefined);
                }}
              >
                {sort === "asc" ? "Ascending" : "Descending"}
              </Button>
              <Button
                variant="outlined"
                color="primary"
                startIcon={<ArrowDownwardIcon />}
                onClick={() => {
                  window.scrollTo({
                    top: document.body.scrollHeight,
                    behavior: "smooth",
                  });
                }}
              >
                Scroll down
              </Button>
            </Box>
          </Box>
          <Box
            component="code"
            sx={{
              mt: 2,
              p: 0,
              display: "block",
              border: `1px solid ${grey[900]}`,
              whiteSpace: whitespace ? "pre-wrap" : undefined,
              wordBreak: whitespace ? "break-all" : undefined,
              fontFamily: "monospace",
              fontSize: 12,
              color: "inherit",
              background: "transparent",
            }}
          >
            {logs.map(renderLog)}
          </Box>
        </>
      ) : (
        <EmptyPage>
          It's quite empty in here.
          <br />
          Logs captured by serverless functions will be displayed here.
        </EmptyPage>
      )}

      {!loading && hasNextPage && (
        <CardFooter sx={{ textAlign: "center" }}>
          <Button
            type="button"
            variant="text"
            loading={loading}
            onClick={() => {
              setKeySeyId(logs[logs.length - 1]?.id);
              setResetLogs(false);
            }}
          >
            Load more
          </Button>
        </CardFooter>
      )}
    </Card>
  );
};

export default RuntimeLogs;
