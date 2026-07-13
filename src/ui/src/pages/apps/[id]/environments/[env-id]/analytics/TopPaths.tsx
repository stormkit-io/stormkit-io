import { useState } from "react";
import Box from "@mui/material/Box";
import Alert from "@mui/material/Alert";
import Typography from "@mui/material/Typography";
import IconButton from "@mui/material/IconButton";
import LinearProgress from "@mui/material/LinearProgress";
import ArrowForward from "@mui/icons-material/ArrowForwardIos";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardRow from "~/components/CardRow";
import CardFooter from "~/components/CardFooter";
import { useFetchTopPaths, useFetchTopReferrers } from "./actions";
import { truncate } from "./helpers";

interface Props {
  environment: Environment;
  domain: Domain;
}

export default function TopPaths({ environment, domain }: Props) {
  const envId = environment.id!;
  const [requestPath, setRequestPath] = useState("");
  const { paths, error, loading } = useFetchTopPaths({
    envId,
    domainId: domain?.id,
  });

  const {
    referrers,
    error: refsError,
    loading: refsLoading,
  } = useFetchTopReferrers({
    envId,
    domainId: domain?.id,
    requestPath,
    skip: requestPath == "",
  });

  return (
    <Card
      sx={{
        width: { xs: "100%", md: "50%" },
        margin: "",
        display: "flex",
        flexDirection: "column",
      }}
      error={error || refsError}
      loading={loading}
    >
      <CardHeader
        title="Paths"
        subtitle="Top 50 visited paths in the last 30 days."
      />
      <Box
        sx={{
          maxHeight: "300px",
          overflow: "auto",
        }}
      >
        {!requestPath ? (
          paths.map(path => (
            <CardRow
              key={path.name}
              chipLabel={
                <Typography component="span">
                  {path.count.toString()}
                </Typography>
              }
              actions={
                <IconButton
                  sx={{ ml: 2 }}
                  size="small"
                  onClick={() => {
                    setRequestPath(path.name);
                  }}
                >
                  <ArrowForward sx={{ fontSize: 12 }} />
                </IconButton>
              }
            >
              <Typography component="span">{truncate(path.name)}</Typography>
            </CardRow>
          ))
        ) : (
          <Box sx={{ flex: 1 }}>
            <CardRow
              actions={
                <IconButton
                  sx={{ ml: 1, textAlign: "center" }}
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
            {refsLoading && <LinearProgress color="secondary" />}
            {!refsLoading &&
              (referrers.length ? (
                referrers.map(ref => (
                  <CardRow
                    key={ref.name}
                    chipLabel={
                      <Typography component="span">
                        {ref.count.toString()}
                      </Typography>
                    }
                  >
                    <Typography component="span">
                      {truncate(ref.name)}
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
                  <Box>No referrers found for this path.</Box>
                </Alert>
              ))}
          </Box>
        )}
      </Box>
      <CardFooter sx={{ textAlign: "left" }}>
        <Typography sx={{ fontSize: 12 }}>
          Client-side routing is not calculated.
        </Typography>
      </CardFooter>
    </Card>
  );
}
